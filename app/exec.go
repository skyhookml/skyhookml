package app

import (
	"../skyhook"

	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// Run this node.
type ExecRunOptions struct {
	// If force, we run even if outputs were already available.
	Force bool

	// If set, limit execution to these keys.
	// Only supported by incremental ops.
	LimitOutputKeys map[string]bool
}
func (node *DBExecNode) Run(opts ExecRunOptions) error {
	// create datasets for this op if needed
	outputDatasets, outputsOK := node.GetDatasets(true)
	if outputsOK && !opts.Force {
		return nil
	}
	for _, ds := range outputDatasets {
		// TODO: for now we clear the output datasets before running
		// but in the future, ops may support incremental execution
		ds.Clear()
	}
	skOutputDatasets := make(map[string]skyhook.Dataset)
	for name, ds := range outputDatasets {
		skOutputDatasets[name] = ds.Dataset
	}

	// get parent datasets
	// for ExecNode parents, get computed dataset
	// in the future, we may need some recursive execution
	parentDatasets := make(map[string][]*DBDataset)
	for name, plist := range node.GetParents() {
		parentDatasets[name] = make([]*DBDataset, len(plist))
		for i, parent := range plist {
			if parent.Type == "n" {
				n := GetExecNode(parent.ID)
				dsList, _ := n.GetDatasets(false)
				if dsList[parent.Name] == nil {
					return fmt.Errorf("dataset for parent node %s[%s] is missing", n.Name, parent.Name)
				}
				parentDatasets[name][i] = dsList[parent.Name]
			} else {
				parentDatasets[name][i] = GetDataset(parent.ID)
			}
		}
	}

	// get items in parent datasets
	items := make(map[string][][]skyhook.Item)
	for name, dslist := range parentDatasets {
		items[name] = make([][]skyhook.Item, len(dslist))
		for i, ds := range dslist {
			var skItems []skyhook.Item
			for _, item := range ds.ListItems() {
				skItems = append(skItems, item.Item)
			}
			items[name][i] = skItems
		}
	}

	// get tasks
	opImpl := skyhook.GetExecOpImpl(node.Op)
	tasks, err := opImpl.GetTasks(Config.CoordinatorURL, node.ExecNode, items)
	if err != nil {
		return err
	}

	// limit tasks to LimitOutputKeys if needed
	if opts.LimitOutputKeys != nil {
		var ntasks []skyhook.ExecTask
		for _, task := range tasks {
			if !opts.LimitOutputKeys[task.Key] {
				continue
			}
			ntasks = append(ntasks, task)
		}
		tasks = ntasks
	}

	// prepare op
	log.Printf("[exec-node %s] [run] acquiring worker", node.Name)
	workerURL := AcquireWorker()
	log.Printf("[exec-node %s] [run] ... acquired worker at %s", node.Name, workerURL)
	defer ReleaseWorker(workerURL)

	beginRequest := skyhook.ExecBeginRequest{
		Node: node.ExecNode,
		OutputDatasets: skOutputDatasets,
	}
	var beginResponse skyhook.ExecBeginResponse
	if err := skyhook.JsonPost(workerURL, "/exec/start", beginRequest, &beginResponse); err != nil {
		return err
	}
	defer func() {
		err := skyhook.JsonPost(workerURL, "/end", skyhook.EndRequest{beginResponse.UUID}, nil)
		if err != nil {
			log.Printf("[exec-node %s] [run] error ending exec container: %v", node.Name, err)
		}
	}()

	nthreads := beginResponse.Parallelism
	log.Printf("[exec-node %s] [run] running %d tasks in %d threads", node.Name, len(tasks), nthreads)

	counter := 0
	var applyErr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < nthreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// get next task
				mu.Lock()
				if counter >= len(tasks) || applyErr != nil {
					mu.Unlock()
					break
				}
				task := tasks[counter]
				counter++
				mu.Unlock()

				log.Printf("[exec-node %s] [run] apply on %s", node.Name, task.Key)
				err := skyhook.JsonPost(beginResponse.BaseURL, "/exec/task", skyhook.ExecTaskRequest{task}, nil)

				if err != nil {
					mu.Lock()
					applyErr = err
					mu.Unlock()
					break
				}
			}
		}()
	}
	wg.Wait()

	if applyErr != nil {
		return applyErr
	}

	log.Printf("[exec-node %s] [run] done", node.Name)
	return nil
}

// Get some number of incremental outputs from this node.
func (node *DBExecNode) Incremental(n int) error {
	log.Printf("[exec-node %s] [incremental] begin execution of %d outputs", node.Name, n)
	// identify all non-incremental ancestors of this node
	// but stop the search at ExecNodes whose outputs have already been computed
	// we will need to run these ancestors in their entirety
	var nonIncremental []Node
	incrementalNodes := make(map[int]*DBExecNode)
	q := []Node{node}
	seen := map[string]bool{node.GraphID(): true}
	for len(q) > 0 {
		cur := q[len(q)-1]
		q = q[0:len(q)-1]

		if cur.IsDone() {
			continue
		}

		if cur.GraphType() != "exec" {
			// all non-exec are non-incremental
			nonIncremental = append(nonIncremental, cur)
			continue
		}

		execNode := cur.(*DBExecNode)
		if !skyhook.GetExecOpImpl(execNode.Op).Incremental {
			nonIncremental = append(nonIncremental, cur)
			continue
		}

		incrementalNodes[execNode.ID] = execNode

		for _, parent := range cur.GraphParents() {
			if seen[parent.GraphID()] {
				continue
			}
			seen[parent.GraphID()] = true
			q = append(q, parent)
		}
	}

	if len(nonIncremental) > 0 {
		log.Printf("[exec-node %s] [incremental] running %d non-incremental ancestors: %v", node.Name, len(nonIncremental), nonIncremental)
		for _, cur := range nonIncremental {
			RunTree(cur)
		}
	}

	// find the output keys for the current node
	computedOutputKeys := make(map[int][]string)
	getKeys := func(parent skyhook.ExecParent) ([]string, bool) {
		if parent.Type == "d" {
			items := GetDataset(parent.ID).ListItems()
			var keys []string
			for _, item := range items {
				keys = append(keys, item.Key)
			}
			return keys, true
		} else if parent.Type == "n" {
			node := GetExecNode(parent.ID)
			if node.IsDone() {
				datasets, _ := node.GetDatasets(false)
				var keys []string
				for _, item := range datasets[parent.Name].ListItems() {
					keys = append(keys, item.Key)
				}
				return keys, true
			} else if computedOutputKeys[node.ID] != nil {
				return computedOutputKeys[node.ID], true
			} else {
				return nil, false
			}
		}
		panic(fmt.Errorf("bad parent type %s", parent.Type))
	}
	for computedOutputKeys[node.ID] == nil {
		for _, cur := range incrementalNodes {
			if computedOutputKeys[cur.ID] != nil {
				continue
			}
			inputs := make(map[string][][]string)
			ready := true
			for name, plist := range cur.GetParents() {
				inputs[name] = make([][]string, len(plist))
				for i, parent := range plist {
					keys, ok := getKeys(parent)
					if !ok {
						ready = false
						break
					}
					inputs[name][i] = keys
				}
			}
			if !ready {
				continue
			}
			outputKeys := skyhook.GetExecOpImpl(cur.Op).GetOutputKeys(cur.ExecNode, inputs)
			if outputKeys == nil {
				outputKeys = []string{}
			}
			computedOutputKeys[cur.ID] = outputKeys
		}
	}

	// what output keys do we want to produce at the last node?
	allKeys := computedOutputKeys[node.ID]
	wantedKeys := make(map[string]bool)
	if len(allKeys) < n {
		n = len(allKeys)
	}
	for _, idx := range rand.Perm(len(allKeys))[0:n] {
		wantedKeys[allKeys[idx]] = true
	}
	log.Printf("[exec-node %s] [incremental] determined %d keys to produce at this node", node.Name, len(wantedKeys))

	// determine which output keys we need to produce at each incremental node
	// to do this, we iteratively propagate needed keys from children to parents until it is stable
	neededOutputKeys := make(map[int]map[string]bool)
	for _, cur := range incrementalNodes {
		neededOutputKeys[cur.ID] = make(map[string]bool)
	}
	neededOutputKeys[node.ID] = wantedKeys
	getNeededOutputsList := func(id int) []string {
		var s []string
		for key := range neededOutputKeys[id] {
			s = append(s, key)
		}
		return s
	}
	for {
		changed := false
		for _, cur := range incrementalNodes {
			neededInputs := skyhook.GetExecOpImpl(cur.Op).GetNeededInputs(cur.ExecNode, getNeededOutputsList(cur.ID))
			for name, plist := range cur.GetParents() {
				for i, parent := range plist {
					if parent.Type != "n" {
						continue
					}
					if incrementalNodes[parent.ID] == nil {
						continue
					}
					for _, key := range neededInputs[name][i] {
						if neededOutputKeys[parent.ID][key] {
							continue
						}
						changed = true
						neededOutputKeys[parent.ID][key] = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	// now we know which output keys we need to compute at every node
	// so let's go ahead and compute them
	nodesDone := make(map[int]bool)
	for !nodesDone[node.ID] {
		for _, cur := range incrementalNodes {
			ready := true
			for _, plist := range cur.GetParents() {
				for _, parent := range plist {
					if parent.Type != "n" {
						continue
					}
					if incrementalNodes[parent.ID] == nil || nodesDone[parent.ID] {
						continue
					}
					ready = false
					break
				}
			}
			if !ready {
				continue
			}

			curOutputKeys := neededOutputKeys[cur.ID]
			log.Printf("[exec-node %s] [incremental] computing %d output keys at node %s", node.Name, len(curOutputKeys), cur.Name)
			err := cur.Run(ExecRunOptions{
				LimitOutputKeys: curOutputKeys,
			})
			if err != nil {
				return err
			}
			nodesDone[cur.ID] = true
		}
	}

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
		node := NewExecNode(request.Name, request.Op, request.Params, request.Inputs, request.Outputs, request.Parents, request.Workspace)
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
			err := node.Run(ExecRunOptions{Force: true})
			if err != nil {
				log.Printf("[exec node %s] run error: %v", node.Name, err)
			}
		}()
	}).Methods("POST")

	Router.HandleFunc("/exec-nodes/{node_id}/incremental", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		r.ParseForm()
		count := skyhook.ParseInt(r.PostForm.Get("count"))

		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		go func() {
			err := node.Incremental(count)
			if err != nil {
				log.Printf("[exec node %s] incremental run error: %v", node.Name, err)
			}
		}()
	}).Methods("POST")
}
