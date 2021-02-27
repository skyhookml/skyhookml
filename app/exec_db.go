package app

import (
	"../skyhook"

	"fmt"
)

type DBExecNode struct {
	skyhook.ExecNode
	Workspace string
}

const ExecNodeQuery = "SELECT id, name, op, params, inputs, outputs, parents, workspace FROM exec_nodes"

func execNodeListHelper(rows *Rows) []*DBExecNode {
	nodes := []*DBExecNode{}
	for rows.Next() {
		var node DBExecNode
		var inputsRaw, outputsRaw, parentsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &inputsRaw, &outputsRaw, &parentsRaw, &node.Workspace)
		node.Inputs = skyhook.ParseExecInputs(inputsRaw)
		node.Outputs = skyhook.ParseExecOutputs(outputsRaw)
		node.Parents = skyhook.ParseExecParents(parentsRaw)


		// make sure parents list is the same length as inputs
		for len(node.Parents) < len(node.Inputs) {
			node.Parents = append(node.Parents, []skyhook.ExecParent{})
		}


		nodes = append(nodes, &node)
	}
	return nodes
}

func ListExecNodes() []*DBExecNode {
	rows := db.Query(ExecNodeQuery)
	return execNodeListHelper(rows)
}

func (ws DBWorkspace) ListExecNodes() []*DBExecNode {
	rows := db.Query(ExecNodeQuery + " WHERE workspace = ?", ws)
	return execNodeListHelper(rows)
}

func GetExecNode(id int) *DBExecNode {
	rows := db.Query(ExecNodeQuery + " WHERE id = ?", id)
	nodes := execNodeListHelper(rows)
	if len(nodes) == 1 {
		return nodes[0]
	} else {
		return nil
	}
}

func NewExecNode(name string, op string, params string, inputs []skyhook.ExecInput, outputs []skyhook.ExecOutput, parents [][]skyhook.ExecParent, workspace string) *DBExecNode {
	res := db.Exec(
		"INSERT INTO exec_nodes (name, op, params, inputs, outputs, parents, workspace) VALUES (?, ?, ?, ?, ?, ?, ?)",
		name, op, params,
		skyhook.ExecInputsToString(inputs), skyhook.ExecOutputsToString(outputs), skyhook.ExecParentsToString(parents),
		workspace,
	)
	node := GetExecNode(res.LastInsertId())
	node.updateOutputs()
	return node
}

func (node *DBExecNode) DatasetRefs() []int {
	var ds []int
	rows := db.Query("SELECT dataset_id FROM exec_ds_refs WHERE node_id = ?", node.ID)
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ds = append(ds, id)
	}
	return ds
}

// Get datasets for each output of this node.
// If create=true, creates new datasets to cover missing ones.
// Also returns true if all datasets already existed.
func (node *DBExecNode) GetDatasets(create bool) (map[string]*DBDataset, bool) {
	// get the old exec-dataset references
	// we'll need to update datasets that are no longer referenced
	existingDS := node.DatasetRefs()

	// find datasets that match current hash
	nodeHash := node.Hash()
	datasets := make(map[string]*DBDataset)
	dsIDSet := make(map[int]bool)
	ok := true
	for _, output := range node.Outputs {
		dsName := fmt.Sprintf("%s[%s]", node.Name, output.Name)
		curHash := fmt.Sprintf("%s[%s]", nodeHash, output.Name)
		ds := FindDataset(curHash)
		if ds == nil {
			ok = false
			if create {
				ds = NewDataset(dsName, "computed", output.DataType, &curHash)
			}
		}

		if ds != nil {
			ds.AddExecRef(node.ID)
			datasets[output.Name] = ds
			dsIDSet[ds.ID] = true
		} else {
			datasets[output.Name] = nil
		}
	}

	// remove references
	for _, id := range existingDS {
		if dsIDSet[id] {
			continue
		}
		GetDataset(id).DeleteExecRef(node.ID)
	}

	return datasets, ok
}

// delete parent references to a node when the node is deleted or its outputs have changed
func DeleteBrokenReferences(node *DBExecNode, newOutputs []skyhook.ExecOutput) {
	outputSet := make(map[string]bool)
	for _, output := range newOutputs {
		outputSet[output.Name] = true
	}
	for _, other := range ListExecNodes() {
		needsUpdate := false
		for _, plist := range other.Parents {
			for _, parent := range plist {
				if parent.Type == "n" && parent.ID == node.ID && !outputSet[parent.Name] {
					needsUpdate = true
					break
				}
			}
		}
		if !needsUpdate {
			continue
		}
		newParents := make([][]skyhook.ExecParent, len(other.Parents))
		for i, plist := range other.Parents {
			for _, parent := range plist {
				if parent.Type == "n" && parent.ID == node.ID && !outputSet[parent.Name] {
					continue
				}
				newParents[i] = append(newParents[i], parent)
			}
		}
		other.Update(ExecNodeUpdate{
			Parents: &newParents,
		})
	}
}

func (node *DBExecNode) updateOutputs() {
	// make sure the current expected outputs match the actual outputs
	impl := skyhook.GetExecOpImpl(node.Op)
	if impl.GetOutputs != nil {
		expectedOutputs := impl.GetOutputs(Config.CoordinatorURL, node.ExecNode)
		if skyhook.ExecOutputsToString(expectedOutputs) != skyhook.ExecOutputsToString(node.Outputs) {
			db.Exec("UPDATE exec_nodes SET outputs = ? WHERE id = ?", skyhook.ExecOutputsToString(expectedOutputs), node.ID)
			DeleteBrokenReferences(node, expectedOutputs)
			node.Outputs = expectedOutputs
		}
	}
}

type ExecNodeUpdate struct {
	Name *string
	Op *string
	Params *string
	Inputs *[]skyhook.ExecInput
	Outputs *[]skyhook.ExecOutput
	Parents *[][]skyhook.ExecParent
}

func (node *DBExecNode) Update(req ExecNodeUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE exec_nodes SET name = ? WHERE id = ?", *req.Name, node.ID)
		node.Name = *req.Name
	}
	if req.Op != nil {
		db.Exec("UPDATE exec_nodes SET op = ? WHERE id = ?", *req.Op, node.ID)
		node.Op = *req.Op
	}
	if req.Params != nil {
		db.Exec("UPDATE exec_nodes SET params = ? WHERE id = ?", *req.Params, node.ID)
		node.Params = *req.Params
	}
	if req.Inputs != nil {
		db.Exec("UPDATE exec_nodes SET inputs = ? WHERE id = ?", skyhook.ExecInputsToString(*req.Inputs), node.ID)
		node.Inputs = *req.Inputs
	}
	if req.Outputs != nil {
		db.Exec("UPDATE exec_nodes SET outputs = ? WHERE id = ?", skyhook.ExecOutputsToString(*req.Outputs), node.ID)
		DeleteBrokenReferences(node, *req.Outputs)
		node.Outputs = *req.Outputs
	}
	if req.Parents != nil {
		db.Exec("UPDATE exec_nodes SET parents = ? WHERE id = ?", skyhook.ExecParentsToString(*req.Parents), node.ID)
		node.Parents = *req.Parents
	}

	node.updateOutputs()
}

func (node *DBExecNode) Delete() {
	dsIDs := node.DatasetRefs()
	for _, id := range dsIDs {
		GetDataset(id).DeleteExecRef(node.ID)
	}

	// remove reference to this node from children
	DeleteBrokenReferences(node, nil)

	db.Exec("DELETE FROM exec_nodes WHERE id = ?", node.ID)
}
