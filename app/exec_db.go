package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
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
		ds.AddExecRef(node.ID)
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

// Delete ExecParent references that match with an isDeleted function.
func DeleteBrokenReferences(isDeleted func(parent skyhook.ExecParent) bool) {
	for _, node := range ListExecNodes() {
		needsUpdate := false
		for _, plist := range node.Parents {
			for _, parent := range plist {
				if isDeleted(parent) {
					needsUpdate = true
					break
				}
			}
		}
		if !needsUpdate {
			continue
		}
		// At least one parent of this node is deleted.
		// So we need to commit an update.
		newParents := make(map[string][]skyhook.ExecParent, len(node.Parents))
		for name, plist := range node.Parents {
			for _, parent := range plist {
				if isDeleted(parent) {
					log.Printf("[exec_db] deleting broken reference from exec node %s to %v", node.Name, parent)
					continue
				}
				newParents[name] = append(newParents[name], parent)
			}
		}
		node.Update(ExecNodeUpdate{
			Parents: &newParents,
		})
	}
}

// Delete broken ExecParent references to a node when the node is deleted or its outputs have changed.
func DeleteReferencesToNode(node *DBExecNode, newOutputs []skyhook.ExecOutput) {
	outputSet := make(map[string]bool)
	for _, output := range newOutputs {
		outputSet[output.Name] = true
	}
	DeleteBrokenReferences(func(parent skyhook.ExecParent) bool {
		return parent.Type == "n" && parent.ID == node.ID && !outputSet[parent.Name]
	})
}

// Delete broken ExecParent references to a dataset that has been deleted.
func DeleteReferencesToDataset(dataset *DBDataset) {
	DeleteBrokenReferences(func(parent skyhook.ExecParent) bool {
		return parent.Type == "d" && parent.ID == dataset.ID
	})
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
	DeleteReferencesToNode(node, node.GetOutputs())
}

func (node *DBExecNode) Delete() {
	dsIDs := node.DatasetRefs()
	for _, id := range dsIDs {
		GetDataset(id).DeleteExecRef(node.ID)
	}

	// remove reference to this node from children
	DeleteReferencesToNode(node, nil)

	db.Exec("DELETE FROM exec_nodes WHERE id = ?", node.ID)
}

// Resolves an ExecParent to a dataset.
// If the dataset is unavailable, returns an error.
func ExecParentToDataset(parent skyhook.ExecParent) (*DBDataset, error) {
	if parent.Type == "d" {
		ds := GetDataset(parent.ID)
		if ds == nil {
			return nil, fmt.Errorf("no dataset found with the specified ID")
		}
		return ds, nil
	} else if parent.Type == "n" {
		otherNode := GetExecNode(parent.ID)
		outputDatasets, _ := otherNode.GetDatasets(false)
		ds := outputDatasets[parent.Name]
		if ds == nil {
			return nil, fmt.Errorf("node %s has no output named %s", otherNode.Name, parent.Name)
		}
		return ds, nil
	}
	return nil, fmt.Errorf("unknown parent type %s", parent.Type)
}
