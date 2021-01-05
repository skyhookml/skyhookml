package app

import (
	"../skyhook"

	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Run this node.
// If force, we run even if outputs were already available.
func (node *DBExecNode) Run(force bool) error {
	// create datasets for this op if needed
	outputDatasets, outputsOK := node.GetDatasets(true)
	if outputsOK && !force {
		return nil
	}
	for _, ds := range outputDatasets {
		// TODO: for now we clear the output datasets before running
		// but in the future, ops may support incremental execution
		ds.Clear()
	}

	// get parent datasets
	// for ExecNode parents, get computed dataset
	// in the future, we may need some recursive execution
	var allParents []skyhook.ExecParent
	allParents = append(allParents, node.Parents...)
	allParents = append(allParents, node.FilterParents...)
	parentDatasets := make([]*DBDataset, len(allParents))
	for i, parent := range allParents {
		if parent.Type == "n" {
			n := GetExecNode(parent.ID)
			dsList, _ := n.GetDatasets(false)
			if dsList[parent.Index] == nil {
				return fmt.Errorf("dataset for parent node %s[%d] is missing", n.Name, parent.Index)
			}
			parentDatasets[i] = dsList[parent.Index]
		} else {
			parentDatasets[i] = GetDataset(parent.ID)
		}
	}

	// get all unique keys in parent datasets
	keys := make(map[string][]skyhook.Item)
	for i, ds := range parentDatasets {
		curKeys := make(map[string]skyhook.Item)
		for _, item := range ds.ListItems() {
			curKeys[item.Key] = item.Item
		}

		// remove previous keys not in this dataset
		for key := range keys {
			if _, ok := curKeys[key]; !ok {
				delete(keys, key)
			}
		}

		// if not filter parent, add to the items
		if i >= len(node.Parents) {
			continue
		}

		for key, item := range curKeys {
			if i > 0 && keys[key] == nil {
				continue
			}
			keys[key] = append(keys[key], item)
		}
	}

	log.Printf("[exec-node %s] [run] got %d unique keys from parents", node.Name, len(keys))

	// prepare op
	opImpl := skyhook.GetExecOpImpl(node.Op)
	op, err := opImpl.Prepare("http://127.0.0.1:8080", node.ExecNode)
	if err != nil {
		return err
	}
	defer op.Close()

	// apply op on each key
	for key, items := range keys {
		log.Printf("[exec-node %s] [run] apply on %s", node.Name, key)
		output, err := op.Apply(key, items)
		if err != nil {
			return err
		}
		for outKey, datas := range output {
			for i := range datas {
				outputDatasets[i].WriteItem(outKey, datas[i])
			}
		}
	}

	log.Printf("[exec-node %s] [run] done", node.Name)

	return nil
}

func init() {
	Router.HandleFunc("/exec-nodes", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		wsName := r.Form.Get("ws")
		if wsName == "" {
			skyhook.JsonResponse(w, ListExecNodes())
		} else {
			ws := GetWorkspace(wsName)
			skyhook.JsonResponse(w, ws.ListExecNodes())
		}
	}).Methods("GET")

	Router.HandleFunc("/exec-nodes", func(w http.ResponseWriter, r *http.Request) {
		var request DBExecNode
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		node := NewExecNode(request.Name, request.Op, request.Params, request.Parents, request.FilterParents, request.DataTypes, request.Workspace)
		skyhook.JsonResponse(w, node)
	}).Methods("POST")

	Router.HandleFunc("/exec-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		skyhook.JsonResponse(w, node)
	}).Methods("GET")

	Router.HandleFunc("/exec-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}

		var request ExecNodeUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		node.Update(request)
	}).Methods("POST")

	Router.HandleFunc("/exec-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		node.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/exec-nodes/{node_id}/datasets", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		datasets, _ := node.GetDatasets(false)
		skyhook.JsonResponse(w, datasets)
	}).Methods("GET")

	Router.HandleFunc("/exec-nodes/{node_id}/run", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		go func() {
			err := node.Run(true)
			if err != nil {
				log.Printf("[exec node %s] run error: %v", node.Name, err)
			}
		}()
	}).Methods("POST")
}
