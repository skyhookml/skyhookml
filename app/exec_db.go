package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"strings"
)

type DBExecNode struct {
	skyhook.ExecNode
	Workspace string
}

const ExecNodeQuery = "SELECT id, name, op, params, parents, workspace FROM exec_nodes"

func execNodeListHelper(rows *Rows) []*DBExecNode {
	nodes := []*DBExecNode{}
	for rows.Next() {
		var node DBExecNode
		var parentsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsRaw, &node.Workspace)
		skyhook.JsonUnmarshal([]byte(parentsRaw), &node.Parents)
		if node.Parents == nil {
			node.Parents = make(map[string][]skyhook.ExecParent)
		}

		// make sure parents list is set for each input
		for _, input := range node.GetInputs() {
			if node.Parents[input.Name] != nil {
				continue
			}
			node.Parents[input.Name] = []skyhook.ExecParent{}
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

func NewExecNode(name string, op string, params string, parents map[string][]skyhook.ExecParent, workspace string) *DBExecNode {
	res := db.Exec(
		"INSERT INTO exec_nodes (name, op, params, parents, workspace) VALUES (?, ?, ?, ?, ?)",
		name, op, params,
		string(skyhook.JsonMarshal(parents)),
		workspace,
	)
	node := GetExecNode(res.LastInsertId())
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
// Also returns bool, which is true if all datasets exist.
func (node *DBExecNode) GetDatasets(create bool) (map[string]*DBDataset, bool) {
	nodeHash := node.Hash()

	// remove references to datasets that don't even start with the nodeHash
	existingDS := node.DatasetRefs()
	for _, id := range existingDS {
		ds := GetDataset(id)
		if !strings.HasPrefix(*ds.Hash, nodeHash) {
			ds.DeleteExecRef(node.ID)
		}
	}

	// find datasets that match current hash
	datasets := make(map[string]*DBDataset)
	ok := true
	for _, output := range node.GetOutputs() {
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
		} else {
			datasets[output.Name] = nil
		}
	}

	return datasets, ok
}

// Get dataset for a virtual node that comes from this node.
// If the datasets don't exist already, we create them.
func (node *DBExecNode) GetVirtualDatasets(vnode *skyhook.VirtualNode) map[string]*DBDataset {
	nodeHash := node.Hash()
	datasets := make(map[string]*DBDataset)

	for _, output := range vnode.GetOutputs() {
		dsName := fmt.Sprintf("%s.%s[%s]", node.Name, vnode.VirtualKey, output.Name)
		curHash := fmt.Sprintf("%s.%s[%s]", nodeHash, vnode.VirtualKey, output.Name)
		ds := FindDataset(curHash)
		if ds == nil {
			ds = NewDataset(dsName, "computed", output.DataType, &curHash)
		}
		datasets[output.Name] = ds
	}
	return datasets
}

// Returns true if all the output datasets are done.
func (node *DBExecNode) IsDone() bool {
	datasets, ok := node.GetDatasets(false)
	if !ok {
		return false
	}
	for _, ds := range datasets {
		if !ds.Done {
			return false
		}
	}
	return true
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
		newParents := make(map[string][]skyhook.ExecParent, len(other.Parents))
		for name, plist := range other.Parents {
			for _, parent := range plist {
				if parent.Type == "n" && parent.ID == node.ID && !outputSet[parent.Name] {
					continue
				}
				newParents[name] = append(newParents[name], parent)
			}
		}
		other.Update(ExecNodeUpdate{
			Parents: &newParents,
		})
	}
}

type ExecNodeUpdate struct {
	Name *string
	Op *string
	Params *string
	Parents *map[string][]skyhook.ExecParent
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
	if req.Parents != nil {
		db.Exec("UPDATE exec_nodes SET parents = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Parents)), node.ID)
		node.Parents = *req.Parents
	}
	DeleteBrokenReferences(node, node.GetOutputs())
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
