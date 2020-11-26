package app

import (
	"../skyhook"

	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (node *DBExecNode) Run() error {
	// get parent datasets
	// for ExecNode parents, get computed dataset
	// in the future, we may need some recursive execution
	parents := make([]*DBDataset, len(node.Parents))
	for i, parent := range node.Parents {
		if parent.Type == "n" {
			n := GetExecNode(parent.ID)
			dsID := n.DatasetIDs[parent.Index]
			if dsID == nil {
				return fmt.Errorf("dataset for parent node %s is missing", n.Name)
			}
			parents[i] = GetDataset(*dsID)
		} else {
			parents[i] = GetDataset(parent.ID)
		}
	}

	// get all unique keys in parent datasets
	keys := make(map[string][]skyhook.Item)
	for _, ds := range parents {
		curKeys := make(map[string]skyhook.Item)
		for _, item := range ds.ListItems() {
			curKeys[item.Key] = item.Item
		}
		for key, item := range curKeys {
			keys[key] = append(keys[key], item)
		}
	}
	for key := range keys {
		if len(keys[key]) < len(parents) {
			delete(keys, key)
		}
	}

	log.Printf("[exec-node %s] [run] got %d unique keys from parents", node.Name, len(keys))

	// create datasets for this op if needed
	node.EnsureDatasets()
	outputDatasets := make([]*DBDataset, len(node.DataTypes))
	for i, id := range node.DatasetIDs {
		outputDatasets[i] = GetDataset(*id)
	}

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
		skyhook.JsonResponse(w, ListExecNodes())
	}).Methods("GET")

	Router.HandleFunc("/exec-nodes", func(w http.ResponseWriter, r *http.Request) {
		var request skyhook.ExecNode
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		node := NewExecNode(request.Name, request.Op, request.Params, request.Parents, request.DataTypes)
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

	Router.HandleFunc("/exec-nodes/{node_id}/run", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		go func() {
			err := node.Run()
			if err != nil {
				log.Printf("[exec node %s] run error: %v", node.Name, err)
			}
		}()
	}).Methods("POST")
}
