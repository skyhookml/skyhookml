package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
)

func (node *DBExecNode) GetGraphID() skyhook.GraphID {
	return skyhook.GraphID{
		Type: "exec",
		ID: node.ID,
	}
}

// Retrieves the Node (VirtualNode or Dataset) based on GraphID.
func GetNodeByGraphID(id skyhook.GraphID) skyhook.Node {
	if id.Type == "exec" {
		if id.VirtualKey != "" {
			panic(fmt.Errorf("addByGraphID called with non-empty VirtualKey"))
		}
		execNode := GetExecNode(id.ID)
		vnode := execNode.GetOp().Virtualize(execNode.ExecNode)
		if id != vnode.GraphID() {
			panic(fmt.Errorf("unexpected id != vnode.GraphID()"))
		}
		return vnode
	} else if id.Type == "dataset" {
		dataset := GetDataset(id.ID)
		return dataset.Dataset
	}
	return nil
}

// Incorporate a subgraph (graph of new nodes) into an existing execution graph.
// The subgraph may present new dependencies that don't exist yet in the graph, so we need to search those.
func IncorporateIntoGraph(graph skyhook.ExecutionGraph, subgraph skyhook.ExecutionGraph) {
	// first, add/replace from subgraph
	for id, node := range subgraph {
		graph[id] = node
	}

	// now perform a search to make sure all ancestors are in the graph
	var q []skyhook.GraphID
	for id := range subgraph {
		q = append(q, id)
	}
	for len(q) > 0 {
		cur := q[len(q)-1]
		q = q[0:len(q)-1]
		for _, parentID := range graph[cur].GraphParents() {
			if graph[parentID] != nil {
				continue
			}
			node := GetNodeByGraphID(parentID)
			graph[node.GraphID()] = node
			q = append(q, node.GraphID())
		}
	}
}

// Build the execution graph rooted at this node.
// Execution graph maps from GraphID to Node.
func (node *DBExecNode) GetGraph() skyhook.ExecutionGraph {
	graph := make(skyhook.ExecutionGraph)
	srcID := node.GetGraphID()
	subgraph := skyhook.ExecutionGraph{srcID: GetNodeByGraphID(srcID)}
	IncorporateIntoGraph(graph, subgraph)
	return graph
}

func (node *DBExecNode) Hash() string {
	graph := node.GetGraph()
	hashes := graph.GetHashStrings()
	return hashes[node.GetGraphID()]
}

// Run the specified node, while running ancestors first if needed.
type RunNodeOptions struct {
	// If force, we run even if outputs were already available.
	Force bool
	// If NoRunTree, we do not run if the parents of targetNode are not available.
	NoRunTree bool
	// MultiExecJobOp to update with jobs for each ExecNode run.
	JobOp *MultiExecJobOp
}
func RunNode(targetNode *DBExecNode, opts RunNodeOptions) error {
	if targetNode.IsDone() && !opts.Force {
		log.Printf("[run-tree %s] this node is already done", targetNode.Name)
		return nil
	}

	log.Printf("[run-tree %s] building graph", targetNode.Name)
	graph := targetNode.GetGraph()
	targetGraphID := targetNode.GetGraphID()

	log.Printf("[run-tree %s] computing ready/needed nodes", targetNode.Name)
	// graph ID to outputs at that node
	// for datasets, the key is always "" (empty string)
	ready := make(map[skyhook.GraphID]map[string]*DBDataset)
	// stores GraphIDs whose outputs not yet available in ready
	missing := make(map[skyhook.GraphID]skyhook.Node)
	// Stores GraphIDs that need to be executed
	// (missing and targetNode depends on it).
	// If a node X is missing but not needed, it implies that there is some
	// intermediate node between X and targetNode whose outputs were computed.
	needed := make(map[skyhook.GraphID]skyhook.Node)
	// map from ExecNode.ID to *DBExecNode instance
	dbExecNodes := make(map[int]*DBExecNode)

	// Re-compute the nodes that are needed.
	// This is called by populateReadyMissing whenever ready/missing are updated.
	recomputeNeeded := func() {
		if missing[targetGraphID] == nil {
			return
		}
		q := []skyhook.GraphID{targetGraphID}
		needed = map[skyhook.GraphID]skyhook.Node{targetGraphID: missing[targetGraphID]}
		for len(q) > 0 {
			cur := q[len(q)-1]
			q = q[0:len(q)-1]

			// add dependencies that are missing but not already in needed
			vnode := needed[cur].(*skyhook.VirtualNode)
			for _, plist := range vnode.Parents {
				for _, vparent := range plist {
					if missing[vparent.GraphID] == nil || needed[vparent.GraphID] != nil {
						continue
					}
					q = append(q, vparent.GraphID)
					needed[vparent.GraphID] = missing[vparent.GraphID]
				}
			}
		}
	}
	// populate ready/missing depending on whether outputs are available
	// also populate dbExecNodes
	populateReadyMissing := func() {
		for graphID, node := range graph {
			if ready[graphID] != nil || missing[graphID] != nil {
				continue
			}
			if graphID.Type == "dataset" {
				// datasets are always ready
				ready[graphID] = map[string]*DBDataset{"": GetDataset(graphID.ID)}
				continue
			}

			if graphID.VirtualKey == "" {
				execNode := GetExecNode(graphID.ID)
				dbExecNodes[graphID.ID] = execNode
				if execNode.IsDone() {
					datasets, ok := execNode.GetDatasets(false)
					if !ok {
						panic(fmt.Errorf("execNode Done but GetDatasets not ok"))
					}
					ready[graphID] = datasets
				} else {
					missing[graphID] = node
				}
			} else {
				execNode := dbExecNodes[graphID.ID]
				datasets := execNode.GetVirtualDatasets(node.(*skyhook.VirtualNode))
				done := true
				for _, ds := range datasets {
					done = done && ds.Done
				}
				if done {
					ready[graphID] = datasets
				} else {
					missing[graphID] = node
				}
			}
		}
		recomputeNeeded()
	}
	if opts.Force {
		missing[targetGraphID] = graph[targetGraphID]
		dbExecNodes[targetNode.ID] = targetNode
	}
	populateReadyMissing()
	log.Printf("[run-tree %s] ... get %d ready, %d missing, %d needed", targetNode.Name, len(ready), len(missing), len(needed))
	if len(needed) != 1 && opts.NoRunTree {
		return fmt.Errorf("NoRunTree is set but more than one node needed")
	}

	// Make sure that we are not going to run multiple conflicting MultiExec jobs.
	// To do so, we:
	// (1) Store the node IDs that we intend to execute in the job metadata.
	// (2) Loop through all other pending MultiExec jobs, and verify that there
	//     are no conflicts.
	// This should always prevent conflicts since we do (1) before (2). In some
	// cases, we may stop all conflicting jobs instead of running one of them; we
	// could avoid that by implementing transactions later, but it's not a big
	// issue.
	conflictErr := func() error {
		// Collect the node IDs we intend to run.
		var ourIDs []int
		ourIDSet := make(map[int]string) // map from node ID to name
		for _, cur := range needed {
			vnode := cur.(*skyhook.VirtualNode)
			origID := vnode.OrigNode.ID
			if ourIDSet[origID] != "" {
				continue
			}
			ourIDs = append(ourIDs, origID)
			ourIDSet[origID] = vnode.OrigNode.Name
		}

		// Commit our node IDs to the database (via job metadata).
		jobID := -1
		if opts.JobOp != nil {
			bytes := skyhook.JsonMarshal(ourIDs)
			opts.JobOp.Job.UpdateMetadata(string(bytes))
			jobID = opts.JobOp.Job.ID
		}

		// Check for conflicts.
		rows := db.Query("SELECT metadata FROM jobs WHERE done = 0 AND type = 'multiexec' AND id != ? AND metadata != ''", jobID)
		for rows.Next() {
			var metadataRaw string
			rows.Scan(&metadataRaw)
			var ids []int
			skyhook.JsonUnmarshal([]byte(metadataRaw), &ids)
			for _, id := range ids {
				if ourIDSet[id] == "" {
					continue
				}
				rows.Close()
				return fmt.Errorf("another job has a conflict on node %s, wait for that job to finish and try again", ourIDSet[id])
			}
		}

		return nil
	}()
	if conflictErr != nil {
		return conflictErr
	}

	// repeatedly run needed nodes where parents are all available
	// until all nodes are done
	for len(needed) > 0 {
		for id, cur := range needed {
			vnode := cur.(*skyhook.VirtualNode)
			// are parents available?
			// also collect the parent datasets here from ready
			avail := true
			parentDatasets := make(map[string][]*DBDataset)
			for name, plist := range vnode.Parents {
				parentDatasets[name] = make([]*DBDataset, len(plist))
				for i, vparent := range plist {
					if ready[vparent.GraphID] == nil {
						avail = false
						break
					}
					parentDatasets[name][i] = ready[vparent.GraphID][vparent.Name]
				}
			}
			if !avail {
				continue
			}

			// enumerate items
			// we need these for Resolve/GetTasks
			parentItems := make(map[string][][]skyhook.Item)
			for name, dslist := range parentDatasets {
				parentItems[name] = make([][]skyhook.Item, len(dslist))
				for i, ds := range dslist {
					var skItems []skyhook.Item
					for _, item := range ds.ListItems() {
						skItems = append(skItems, item.Item)
					}
					parentItems[name][i] = skItems
				}
			}

			// make sure this node doesn't Resolve to something else if needed
			subgraph := vnode.GetOp().Resolve(vnode, ToSkyhookInputDatasets(parentDatasets), parentItems)
			if subgraph != nil {
				// this vnode wants to be dynamically replaced with the new subgraph
				// we need to incorporate the subgraph into our graph
				log.Printf("[run-tree %s] node %s resolved into a subgraph of size %d, adding to our graph of size %d", targetNode.Name, vnode.Name, len(subgraph), len(graph))
				IncorporateIntoGraph(graph, subgraph)
				log.Printf("[run-tree %s] ... graph grew to size %d", targetNode.Name, len(graph))

				// we also need to populate ready and missing with any new nodes
				// so we call populateReadyMissing with anything that's not already there
				populateReadyMissing()

				// parents and stuff may have changed now
				// so we need to re-evaluate whether parent datasets are available
				// so: skip processing for now
				continue
			}

			log.Printf("[run-tree %s] running node %s", targetNode.Name, vnode.Name)

			// get output datasets
			origNode := dbExecNodes[vnode.OrigNode.ID]
			var outputDatasets map[string]*DBDataset
			if vnode.VirtualKey == "" {
				outputDatasets, _ = origNode.GetDatasets(true)
			} else {
				outputDatasets = origNode.GetVirtualDatasets(vnode)
			}
			for _, ds := range outputDatasets {
				ds.Clear()
				ds.SetDone(false)
			}

			// load runnable
			runnable := vnode.GetRunnable(ToSkyhookInputDatasets(parentDatasets), ToSkyhookOutputDatasets(outputDatasets))

			// Initialize job.
			rd := &RunData{
				Name: vnode.Name,
				Node: runnable,
				WillBeDone: true,
			}
			rd.SetJob(fmt.Sprintf("Exec Node %s", vnode.Name), fmt.Sprintf("%d", vnode.OrigNode.ID))
			// if MultiExecJobOp is provided, we need to update it with the current job
			if opts.JobOp != nil {
				opts.JobOp.SetPlanFromGraph(graph, ready, needed, vnode)
				opts.JobOp.ChangeJob(rd.JobOp.Job.Job)
			}

			// Get tasks.
			// We do this after initializing RunData so that we can log any error to the AppJobOp.
			var err error
			rd.Tasks, err = runnable.GetOp().GetTasks(runnable, parentItems)
			if err != nil {
				rd.JobOp.SetDone(err)
				return err
			}

			// Run the node.
			err = rd.Run()
			rd.SetDone()
			if err != nil {
				return err
			}

			delete(needed, id)
			ready[id] = outputDatasets
		}
	}

	// update plan for last time if needed
	if opts.JobOp != nil {
		opts.JobOp.SetPlanFromGraph(graph, ready, needed, nil)
	}

	return nil
}
