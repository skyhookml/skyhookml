package app

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

func ToSkyhookInputDatasets(datasets map[string][]*DBDataset) map[string][]skyhook.Dataset {
	sk := make(map[string][]skyhook.Dataset)
	for name, dslist := range datasets {
		for _, ds := range dslist {
			sk[name] = append(sk[name], ds.Dataset)
		}
	}
	return sk
}

func ToSkyhookOutputDatasets(datasets map[string]*DBDataset) map[string]skyhook.Dataset {
	sk := make(map[string]skyhook.Dataset)
	for name, ds := range datasets {
		sk[name] = ds.Dataset
	}
	return sk
}

// Helper function to compute the keys already computed at a node.
// This only works for incremental nodes, which must produce the same keys across all output datasets.
func (node *DBExecNode) GetComputedKeys() map[string]bool {
	outputDatasets, _ := node.GetDatasets(false)
	outputItems := make(map[string][][]skyhook.Item)
	for name, ds := range outputDatasets {
		if ds == nil {
			return nil
		}
		var skItems []skyhook.Item
		for _, item := range ds.ListItems() {
			skItems = append(skItems, item.Item)
		}
		outputItems[name] = [][]skyhook.Item{skItems}
	}
	groupedItems := exec_ops.GroupItems(outputItems)
	keySet := make(map[string]bool)
	for key := range groupedItems {
		keySet[key] = true
	}
	return keySet
}

type ExecRunOptions struct {
	// If force, we run even if outputs were already available.
	Force bool

	// Whether to try incremental execution at this node.
	// If false, we throw error if parent datasets are not done.
	Incremental bool

	// If set, limit execution to these keys.
	// Only supported by incremental ops.
	LimitOutputKeys map[string]bool
}

// A RunData provides a Run function that executes a Runnable over the specified tasks.
type RunData struct {
	Name string
	Node skyhook.Runnable
	Tasks []skyhook.ExecTask

	// whether we'll be done with the node after running Tasks
	// i.e., whether Tasks contains all pending tasks at this node
	WillBeDone bool

	// job-related things to update
	JobOp *AppJobOp
	ProgressJobOp *ProgressJobOp

	// Saved error if any
	Error error
}

// Create a Job for this RunData and populate JobOp/ProgressJobOp.
func (rd *RunData) SetJob(name string, metadata string) {
	if rd.JobOp != nil {
		return
	}

	// initialize job
	// if the node doesn't provide a custom JobOp, we use "consoleprogress" view
	// otherwise the view for the job is the ExecOp's name
	opImpl := rd.Node.GetOp()
	nodeJobOp, nodeView := opImpl.GetJobOp(rd.Node)
	jobView := "consoleprogress"
	if nodeView != "" {
		jobView = nodeView
	}
	job := NewJob(
		fmt.Sprintf("Exec Node %s", name),
		"execnode",
		jobView,
		metadata,
	)

	rd.ProgressJobOp = &ProgressJobOp{}
	rd.JobOp = &AppJobOp{
		Job: job,
		TailOp: &skyhook.TailJobOp{},
		WrappedJobOps: map[string]skyhook.JobOp{
			"progress": rd.ProgressJobOp,
		},
	}
	if nodeJobOp != nil {
		rd.JobOp.WrappedJobOps["node"] = nodeJobOp
	}
	job.AttachOp(rd.JobOp)
}

// Update the AppJobOp with the saved error.
// We don't call this in RunData.Run by default because it's possible that the
// specified RunData.JobOp is shared across multiple Runs and shouldn't be
// marked as completed.
func (rd *RunData) SetDone() {
	if rd.Error == nil {
		rd.JobOp.SetDone(nil)
	} else {
		rd.JobOp.SetDone(rd.Error)
	}
}

// Prepare to run this node.
// Returns a RunData.
// Or error on error.
// Or nil RunData and error if the node is already done.
func (node *DBExecNode) PrepareRun(opts ExecRunOptions) (*RunData, error) {
	// create datasets for this op if needed
	outputDatasets, _ := node.GetDatasets(true)

	// if force, we clear the datasets first
	// otherwise, check if the datasets are done already
	if opts.Force {
		for _, ds := range outputDatasets {
			ds.Clear()
			ds.SetDone(false)
		}
	} else {
		done := true
		for _, ds := range outputDatasets {
			done = done && ds.Done
		}
		if done {
			return nil, nil
		}
	}

	// get parent datasets
	// for ExecNode parents, get computed dataset
	// in the future, we may need some recursive execution
	parentDatasets := make(map[string][]*DBDataset)
	parentsDone := true // whether parent datasets are fully computed
	for name, plist := range node.Parents {
		parentDatasets[name] = make([]*DBDataset, len(plist))
		for i, parent := range plist {
			if parent.Type == "n" {
				n := GetExecNode(parent.ID)
				dsList, _ := n.GetDatasets(false)
				ds := dsList[parent.Name]
				if ds == nil {
					return nil, fmt.Errorf("dataset for parent node %s[%s] is missing", n.Name, parent.Name)
				} else if !ds.Done && !opts.Incremental {
					return nil, fmt.Errorf("dataset for parent node %s[%s] is not done", n.Name, parent.Name)
				}
				parentDatasets[name][i] = ds
				parentsDone = parentsDone && ds.Done
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
	opImpl := node.GetOp()
	vnode := opImpl.Virtualize(node.ExecNode)
	runnable := vnode.GetRunnable(ToSkyhookInputDatasets(parentDatasets), ToSkyhookOutputDatasets(outputDatasets))
	tasks, err := opImpl.GetTasks(runnable, items)
	if err != nil {
		return nil, err
	}

	// if running incrementally, remove tasks that were already computed
	// this is mostly so that we can see whether we will be done with this node after the current execution
	// (i.e., we are done here if parentsDone and we execute all remaining tasks)
	if opts.Incremental {
		var ntasks []skyhook.ExecTask
		completedKeys := node.GetComputedKeys()
		for _, task := range tasks {
			if completedKeys[task.Key] {
				continue
			}
			ntasks = append(ntasks, task)
		}
		tasks = ntasks
	}

	// limit tasks to LimitOutputKeys if needed
	// also determine whether this current execution will lead to all tasks being completed
	willBeDone := true
	if !parentsDone {
		willBeDone = false
	}
	if opts.LimitOutputKeys != nil {
		var ntasks []skyhook.ExecTask
		for _, task := range tasks {
			if !opts.LimitOutputKeys[task.Key] {
				continue
			}
			ntasks = append(ntasks, task)
		}
		if len(ntasks) != len(tasks) {
			tasks = ntasks
			willBeDone = false
		}
	}

	rd := &RunData{
		Name: node.Name,
		Node: runnable,
		Tasks: tasks,
		WillBeDone: willBeDone,
	}
	rd.SetJob(fmt.Sprintf("Exec Node %s", node.Name), fmt.Sprintf("%d", node.ID))
	return rd, nil
}

func (rd *RunData) Run() error {
	name := rd.Name

	// get container corresponding to rd.Node.Op
	log.Printf("[exec-node %s] [run] acquiring container", name)
	rd.JobOp.Update([]string{"Acquiring worker"})
	if err := AcquireWorker(rd.JobOp); err != nil {
		rd.Error = err
		return err
	}
	defer ReleaseWorker()
	containerInfo, err := AcquireContainer(rd.Node, rd.JobOp)
	if err != nil {
		rd.Error = err
		return err
	}
	log.Printf("[exec-node %s] [run] ... acquired container %s at %s", name, containerInfo.UUID, containerInfo.BaseURL)
	defer ReleaseWorker()

	// we want to de-allocate the container in two cases:
	// (1) when we return from this function
	// (2) if user requests to stop this job
	// we achieve this as follows:
	// - associate cleanup func with the JobOp
	// - on return, call AppJobOp.Cleanup to only de-allocate if it hasn't been de-allocated already
	// this is possible because AppJobOp will take care of unsetting CleanupFunc whenever it's called
	rd.JobOp.SetCleanupFunc(func() {
		err := skyhook.JsonPost(Config.WorkerURL, "/container/end", skyhook.EndRequest{containerInfo.UUID}, nil)
		if err != nil {
			log.Printf("[exec-node %s] [run] error ending exec container: %v", name, err)
		}
	})
	defer rd.JobOp.Cleanup()

	nthreads := containerInfo.Parallelism
	log.Printf("[exec-node %s] [run] running %d tasks in %d threads", name, len(rd.Tasks), nthreads)
	rd.ProgressJobOp.SetTotal(len(rd.Tasks))

	counter := 0
	var applyErr error
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < nthreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !rd.JobOp.IsStopping() {
				// get next task
				mu.Lock()
				if counter >= len(rd.Tasks) || applyErr != nil {
					mu.Unlock()
					break
				}
				task := rd.Tasks[counter]
				counter++
				mu.Unlock()

				log.Printf("[exec-node %s] [run] apply on %s", name, task.Key)
				err := skyhook.JsonPost(containerInfo.BaseURL, "/exec/task", skyhook.ExecTaskRequest{task}, nil)

				if err != nil {
					mu.Lock()
					applyErr = err
					mu.Unlock()
					break
				}

				mu.Lock()
				rd.ProgressJobOp.Increment()
				rd.JobOp.Update([]string{fmt.Sprintf("finished applying on key [%s]", task.Key)})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if applyErr != nil {
		rd.Error = applyErr
		return applyErr
	}

	// update dataset states
	if rd.WillBeDone {
		for _, ds := range rd.Node.OutputDatasets {
			(&DBDataset{Dataset: ds}).SetDone(true)
		}
	}

	log.Printf("[exec-node %s] [run] done", name)
	return nil
}

// Get some number of incremental outputs from this node.
type IncrementalOptions struct {
	// Number of random outputs to compute at this node.
	// Only one of Count or Keys should be specified.
	Count int
	// Compute outputs matching these keys.
	Keys []string
	// MultiExecJob to update during incremental execution.
	// For non-incremental ancestors, we pass this JobOp to RunNode.
	JobOp *MultiExecJobOp
}
func (node *DBExecNode) Incremental(opts IncrementalOptions) error {
	isIncremental := func(node *DBExecNode) bool {
		return node.GetOp().IsIncremental()
	}

	if !isIncremental(node) {
		return fmt.Errorf("can only incrementally run incremental nodes")
	} else if node.IsDone() {
		return nil
	}

	log.Printf("[exec-node %s] [incremental] begin execution", node.Name)
	// identify all non-incremental ancestors of this node
	// but stop the search at ExecNodes whose outputs have already been computed
	// we will need to run these ancestors in their entirety
	// note: we do not need to worry about Virtualize here because we assume Virtualize and Incremental are mutually exclusive
	var nonIncremental []*DBExecNode
	incrementalNodes := make(map[int]*DBExecNode)
	q := []*DBExecNode{node}
	seen := map[int]bool{node.ID: true}
	for len(q) > 0 {
		cur := q[len(q)-1]
		q = q[0:len(q)-1]

		if cur.IsDone() {
			continue
		}

		if !isIncremental(cur) {
			nonIncremental = append(nonIncremental, cur)
			continue
		}

		incrementalNodes[cur.ID] = cur

		for _, plist := range cur.Parents {
			for _, parent := range plist {
				if parent.Type != "n" {
					continue
				}
				if seen[parent.ID] {
					continue
				}
				seen[parent.ID] = true
				parentNode := GetExecNode(parent.ID)
				q = append(q, parentNode)
			}
		}
	}

	if len(nonIncremental) > 0 {
		log.Printf("[exec-node %s] [incremental] running %d non-incremental ancestors: %v", node.Name, len(nonIncremental), nonIncremental)
		for _, cur := range nonIncremental {
			RunNode(cur, RunNodeOptions{
				JobOp: opts.JobOp,
			})
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
			for name, plist := range cur.Parents {
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
			outputKeys := cur.GetOp().GetOutputKeys(cur.ExecNode, inputs)
			if outputKeys == nil {
				outputKeys = []string{}
			}
			computedOutputKeys[cur.ID] = outputKeys
		}
	}

	// what output keys haven't been computed yet at the last node?
	allKeys := computedOutputKeys[node.ID]
	persistedKeys := node.GetComputedKeys()
	var missingKeys []string
	for _, key := range allKeys {
		if persistedKeys[key] {
			continue
		}
		missingKeys = append(missingKeys, key)
	}
	log.Printf("[exec-node %s] [incremental] found %d total keys, %d already computed keys, and %d missing keys at this node", node.Name, len(allKeys), len(persistedKeys), len(missingKeys))

	// what output keys do we want to produce at the last node?
	wantedKeys := make(map[string]bool)
	if opts.Count > 0 {
		n := opts.Count
		if len(missingKeys) < n {
			n = len(missingKeys)
		}
		for _, idx := range rand.Perm(len(missingKeys))[0:n] {
			wantedKeys[missingKeys[idx]] = true
		}
	} else {
		missingSet := make(map[string]bool)
		for _, key := range missingKeys {
			missingSet[key] = true
		}
		for _, key := range opts.Keys {
			if !missingSet[key] {
				continue
			}
			wantedKeys[key] = true
		}
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
			neededInputs := cur.GetOp().GetNeededInputs(cur.ExecNode, getNeededOutputsList(cur.ID))
			for name, plist := range cur.Parents {
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
			if nodesDone[cur.ID] {
				continue
			}

			ready := true
			for _, plist := range cur.Parents {
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

			rd, err := cur.PrepareRun(ExecRunOptions{
				Incremental: true,
				LimitOutputKeys: curOutputKeys,
			})
			if err != nil {
				return err
			}
			if rd == nil {
				// Already done.
				continue
			}

			if opts.JobOp != nil {
				opts.JobOp.SetPlanFromMap(incrementalNodes, nodesDone, cur.ID)
				opts.JobOp.ChangeJob(rd.JobOp.Job.Job)
			}

			err = rd.Run()
			rd.SetDone()
			if err != nil {
				return err
			}

			nodesDone[cur.ID] = true
		}
	}

	return nil
}

func init() {
	type FrontendExecNode struct {
		DBExecNode
		Inputs []skyhook.ExecInput
		Outputs []skyhook.ExecOutput
	}
	getFrontendExecNode := func(node *DBExecNode) FrontendExecNode {
		inputs := node.GetInputs()
		outputs := node.GetOutputs()
		if inputs == nil {
			inputs = []skyhook.ExecInput{}
		}
		if outputs == nil {
			outputs = []skyhook.ExecOutput{}
		}
		return FrontendExecNode{
			DBExecNode: *node,
			Inputs: inputs,
			Outputs: outputs,
		}
	}

	Router.HandleFunc("/exec-nodes", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		wsName := r.Form.Get("ws")
		var execNodes []*DBExecNode
		if wsName == "" {
			execNodes = ListExecNodes()
		} else {
			ws := GetWorkspace(wsName)
			execNodes = ws.ListExecNodes()
		}
		out := make([]FrontendExecNode, len(execNodes))
		for i := range out {
			out[i] = getFrontendExecNode(execNodes[i])
		}
		skyhook.JsonResponse(w, out)
	}).Methods("GET")

	Router.HandleFunc("/exec-nodes", func(w http.ResponseWriter, r *http.Request) {
		var request DBExecNode
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}
		node := NewExecNode(request.Name, request.Op, request.Params, request.Parents, request.Workspace)
		skyhook.JsonResponse(w, node)
	}).Methods("POST")

	Router.HandleFunc("/exec-nodes/{node_id}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		skyhook.JsonResponse(w, getFrontendExecNode(node))
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

		// initialize job for this run
		job := NewJob(
			fmt.Sprintf("Exec Tree %s", node.Name),
			"multiexec",
			"multiexec",
			"",
		)
		jobOp := &MultiExecJobOp{Job: job}
		job.AttachOp(jobOp)

		go func() {
			err := RunNode(node, RunNodeOptions{
				Force: true,
				JobOp: jobOp,
			})
			job.UpdateState(jobOp.Encode())
			if err != nil {
				log.Printf("[exec node %s] run error: %v", node.Name, err)
				job.SetDone(err.Error())
			} else {
				job.SetDone("")
			}
		}()

		skyhook.JsonResponse(w, job)
	}).Methods("POST")

	// Endpoint to mark the outputs of a node as done.
	// It returns an error if the output datasets aren't even created yet.
	// This is provided so that, in some cases, a job can be manually terminated
	// and the user can opt to use the incomplete outputs for the next job.
	// Specifically, it is used in pytorch_train to stop the job and keep the
	// current saved model for downstream operations.
	Router.HandleFunc("/exec-nodes/{node_id}/set-done", func(w http.ResponseWriter, r *http.Request) {
		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}
		datasets, ok := node.GetDatasets(false)
		if !ok {
			http.Error(w, "can't mark outputs done since some outputs do not exist", 400)
			return
		}
		for _, ds := range datasets {
			if !ds.Done {
				log.Printf("[set-done] manually marking dataset %d [%s] as done", ds.ID, ds.Name)
				ds.SetDone(true)
			}
		}
	}).Methods("POST")

	Router.HandleFunc("/exec-nodes/{node_id}/incremental", func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			// One of random, dataset, or direct.
			Mode string
			// Random mode: number of outputs to compute.
			Count int
			// Dataset mode: ExecParent specifying a dataset whose keys we should compute.
			ParentSpec skyhook.ExecParent
			// Direct mode: list of keys to compute.
			Keys []string
		}
		if err := skyhook.ParseJsonRequest(w, r, &params); err != nil {
			return
		}

		nodeID := skyhook.ParseInt(mux.Vars(r)["node_id"])
		node := GetExecNode(nodeID)
		if node == nil {
			http.Error(w, "no such exec node", 404)
			return
		}

		var opts IncrementalOptions
		if params.Mode == "random" {
			opts.Count = params.Count
		} else if params.Mode == "direct" {
			opts.Keys = params.Keys
		} else if params.Mode == "dataset" {
			// If mode is dataset, we need to get the items in the dataset and determine
			// the concrete list of keys that we should compute.
			dataset, err := ExecParentToDataset(params.ParentSpec)
			if err != nil {
				http.Error(w, "could not find the specified dataset; make sure the dataset specifying keys to compute is already computed", 400)
				return
			}
			for _, item := range dataset.ListItems() {
				opts.Keys = append(opts.Keys, item.Key)
			}
		}

		// initialize job for this run
		job := NewJob(
			fmt.Sprintf("Partial Execution %s", node.Name),
			"multiexec",
			"multiexec",
			"",
		)
		jobOp := &MultiExecJobOp{Job: job}
		job.AttachOp(jobOp)
		opts.JobOp = jobOp

		go func() {
			err := node.Incremental(opts)
			job.UpdateState(jobOp.Encode())
			if err != nil {
				log.Printf("[exec node %s] incremental run error: %v", node.Name, err)
				job.SetDone(err.Error())
			} else {
				job.SetDone("")
			}
		}()

		skyhook.JsonResponse(w, job)
	}).Methods("POST")

	// Execution of an anonymous Runnable with arbitrary configuration.
	Router.HandleFunc("/runnable", func(w http.ResponseWriter, r *http.Request) {
		var node skyhook.Runnable
		if err := skyhook.ParseJsonRequest(w, r, &node); err != nil {
			return
		}

		// compute items in input datasets
		items := make(map[string][][]skyhook.Item)
		for name, dslist := range node.InputDatasets {
			items[name] = make([][]skyhook.Item, len(dslist))
			for i, ds_ := range dslist {
				ds := GetDataset(ds_.ID)
				var curItems []skyhook.Item
				for _, item := range ds.ListItems() {
					curItems = append(curItems, item.Item)
				}
				items[name][i] = curItems
			}
		}

		// get tasks
		tasks, err := node.GetOp().GetTasks(node, items)
		if err != nil {
			http.Error(w, err.Error(), 400)
			log.Printf("[/runnable] error getting tasks: %v", err)
			return
		}

		log.Printf("[/runnable] executing anonymous op [%s-%s] on %d tasks", node.Op, node.Name, len(tasks))

		rd := &RunData{
			Name: fmt.Sprintf("anonymous-%s", node.Name),
			Node: node,
			Tasks: tasks,
			WillBeDone: true,
		}
		rd.SetJob(node.Name, "")
		go func() {
			err := rd.Run()
			rd.SetDone()
			if err != nil {
				log.Printf("[/runnable] error on anonymous op [%s-%s]: %v", node.Op, node.Name, err)
			}
		}()

		skyhook.JsonResponse(w, rd.JobOp.Job)
	}).Methods("POST")
}
