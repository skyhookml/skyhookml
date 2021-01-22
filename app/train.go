package app

import (
	"../skyhook"

	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Run this node.
// If force, we run even if trained model already available.
func (node *DBTrainNode) Run(force bool) error {
	nodeHash := node.Hash()

	// check existing model
	if node.ModelID != nil {
		prevModel := GetModel(*node.ModelID)
		if prevModel.Hash == nodeHash && !force {
			return nil
		}
		node.ModelID = nil
		db.Exec("UPDATE train_nodes SET model_id = NULL WHERE id = ?", node.ID)
		prevModel.CheckRefs()
	}


	log.Printf("[train-node %s] [run] acquiring worker", node.Name)
	workerURL := AcquireWorker()
	log.Printf("[train-node %s] [run] ... acquired worker at %s", node.Name, workerURL)
	defer ReleaseWorker(workerURL)

	beginRequest := skyhook.TrainBeginRequest{
		Node: node.TrainNode,
	}
	var beginResponse skyhook.TrainBeginResponse
	if err := skyhook.JsonPost(workerURL, "/train/start", beginRequest, &beginResponse); err != nil {
		return err
	}
	defer func() {
		err := skyhook.JsonPost(workerURL, "/end", skyhook.EndRequest{beginResponse.UUID}, nil)
		if err != nil {
			log.Printf("[train-node %s] [run] error ending train container: %v", node.Name, err)
		}
	}()

	for {
		var pollResponse skyhook.TrainPollResponse
		if err := skyhook.JsonGet(beginResponse.BaseURL, "/train/poll", &pollResponse); err != nil {
			return err
		}
		if !pollResponse.Done {
			time.Sleep(time.Second)
			continue
		}
		if pollResponse.Error != "" {
			return fmt.Errorf(pollResponse.Error)
		}
	}

	res := db.Exec("INSERT INTO models (hash) VALUES (?)", nodeHash)
	modelID := res.LastInsertId()
	node.ModelID = &modelID
	db.Exec("UPDATE train_nodes SET model_id = ? WHERE id = ?", modelID, node.ID)
	return nil
}

func init() {
	Router.HandleFunc("/train-nodes", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		wsName := r.Form.Get("ws")
		if wsName == "" {
			skyhook.JsonResponse(w, ListTrainNodes())
		} else {
			ws := GetWorkspace(wsName)
			skyhook.JsonResponse(w, ws.ListTrainNodes())
		}
	}).Methods("GET")

	Router.HandleFunc("/train-nodes", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		op := r.PostForm.Get("op")
		wsName := r.PostForm.Get("ws")
		node := NewTrainNode(name, op, wsName)
		skyhook.JsonResponse(w, node)
	}).Methods("POST")

	Router.HandleFunc("/train-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetTrainNode(nodeID)
		if node == nil {
			http.Error(w, "no such train node", 404)
			return
		}
		skyhook.JsonResponse(w, node)
	}).Methods("GET")

	Router.HandleFunc("/train-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetTrainNode(nodeID)
		if node == nil {
			http.Error(w, "no such train node", 404)
			return
		}

		var request TrainNodeUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		node.Update(request)
	}).Methods("POST")

	Router.HandleFunc("/train-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetTrainNode(nodeID)
		if node == nil {
			http.Error(w, "no such train node", 404)
			return
		}
		node.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/train-nodes/{node_id}/run", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetTrainNode(nodeID)
		if node == nil {
			http.Error(w, "no such train node", 404)
			return
		}
		go func() {
			err := node.Run(true)
			if err != nil {
				log.Printf("[train node %s] run error: %v", node.Name, err)
			}
		}()
	}).Methods("POST")

	Router.HandleFunc("/pytorch/components", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListPytorchComponents())
	}).Methods("GET")

	Router.HandleFunc("/pytorch/components", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		comp := NewPytorchComponent(name)
		skyhook.JsonResponse(w, comp)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/components/{comp_id}", func(w http.ResponseWriter, r *http.Request) {
		compID := skyhook.ParseInt(mux.Vars(r)["comp_id"])
		comp := GetPytorchComponent(compID)
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}
		skyhook.JsonResponse(w, comp)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/components/{comp_id}", func(w http.ResponseWriter, r *http.Request) {
		compID := skyhook.ParseInt(mux.Vars(r)["comp_id"])
		comp := GetPytorchComponent(compID)
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}

		var request PytorchComponentUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		comp.Update(request)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListPytorchArchs())
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		arch := NewPytorchArch(name)
		skyhook.JsonResponse(w, arch)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/archs/{arch_id}", func(w http.ResponseWriter, r *http.Request) {
		archID := skyhook.ParseInt(mux.Vars(r)["arch_id"])
		arch := GetPytorchArch(archID)
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}
		skyhook.JsonResponse(w, arch)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs/{arch_id}", func(w http.ResponseWriter, r *http.Request) {
		archID := skyhook.ParseInt(mux.Vars(r)["arch_id"])
		arch := GetPytorchArch(archID)
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}

		var request PytorchArchUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		arch.Update(request)
	}).Methods("POST")
}
