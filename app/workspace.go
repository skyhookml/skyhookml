package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type DBWorkspace string

func GetWorkspace(wsName string) *DBWorkspace {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM workspaces WHERE name = ?", wsName).Scan(&count)
	if count == 0 {
		return nil
	} else {
		ws := DBWorkspace(wsName)
		return &ws
	}
}

func (ws DBWorkspace) Delete() {
	for _, node := range ws.ListExecNodes() {
		node.Delete()
	}
	db.Exec("DELETE FROM ws_datasets WHERE workspace = ?", ws)
	db.Exec("DELETE FROM workspaces WHERE name = ?", ws)
}

func init() {
	Router.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		var workspaces []string
		rows := db.Query("SELECT name FROM workspaces")
		for rows.Next() {
			var ws string
			rows.Scan(&ws)
			workspaces = append(workspaces, ws)
		}
		skyhook.JsonResponse(w, workspaces)
	}).Methods("GET")

	Router.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		db.Exec("INSERT INTO workspaces (name) VALUES (?)", name)
	}).Methods("POST")

	Router.HandleFunc("/workspaces/{ws}/clone", func(w http.ResponseWriter, r *http.Request) {
		ws := DBWorkspace(mux.Vars(r)["ws"])
		r.ParseForm()
		cloneWS := r.PostForm.Get("name")

		log.Printf("cloning workspace %v into new workspace %s", ws, cloneWS)

		// create workspace
		db.Exec("INSERT INTO workspaces (name) VALUES (?)", cloneWS)

		// copy exec nodes
		func() {
			pendingNodes := make(map[int]*DBExecNode)
			for _, node := range ws.ListExecNodes() {
				pendingNodes[node.ID] = node
			}
			// map from old node ID to new node object
			newNodes := make(map[int]*DBExecNode)
			for len(pendingNodes) > 0 {
				for id, node := range pendingNodes {
					// collect parents
					getParents := func(oldParents [][]skyhook.ExecParent) ([][]skyhook.ExecParent, bool) {
						parents := make([][]skyhook.ExecParent, len(oldParents))
						for i, plist := range oldParents {
							parents[i] = make([]skyhook.ExecParent, len(plist))
							for j, parent := range plist {
								if parent.Type == "n" {
									if newNodes[parent.ID] == nil {
										return nil, false
									}
									parents[i][j] = skyhook.ExecParent{
										Type: "n",
										ID: newNodes[parent.ID].ID,
										Name: parent.Name,
									}
								} else if parent.Type == "d" {
									parents[i][j] = parent
								}
							}
						}
						return parents, true
					}
					parents, ok := getParents(node.Parents)
					if !ok {
						continue
					}
					node_ := NewExecNode(node.Name, node.Op, node.Params, node.Inputs, node.Outputs, parents, cloneWS)
					newNodes[id] = node_
					delete(pendingNodes, id)
				}
			}
		}()

		// copy datasets
		db.Exec("INSERT INTO ws_datasets (dataset_id, workspace) SELECT dataset_id, ? FROM ws_datasets WHERE workspace = ?", cloneWS, ws)
	}).Methods("POST")

	Router.HandleFunc("/workspaces/{ws}", func(w http.ResponseWriter, r *http.Request) {
		ws := DBWorkspace(mux.Vars(r)["ws"])
		ws.Delete()
	}).Methods("DELETE")
}
