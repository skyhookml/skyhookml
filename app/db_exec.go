package app

import (
	"../skyhook"

	"fmt"
	"strings"
)

type DBExecNode struct {
	skyhook.ExecNode
	Workspace string
}

const ExecNodeQuery = "SELECT id, name, op, params, parents, data_types, workspace FROM exec_nodes"

func execNodeListHelper(rows *Rows) []*DBExecNode {
	nodes := []*DBExecNode{}
	for rows.Next() {
		var node DBExecNode
		var parentsRaw, typesRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsRaw, &typesRaw, &node.Workspace)
		node.Parents = skyhook.ParseExecParents(parentsRaw)
		node.DataTypes = skyhook.DecodeTypes(typesRaw)
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

func NewExecNode(name string, op string, params string, parents []skyhook.ExecParent, dataTypes []skyhook.DataType, workspace string) *DBExecNode {
	res := db.Exec(
		"INSERT INTO exec_nodes (name, op, params, parents, data_types, workspace) VALUES (?, ?, ?, ?, ?, ?, ?)",
		name, op, params, skyhook.ExecParentsToString(parents), skyhook.EncodeTypes(dataTypes), workspace,
	)
	return GetExecNode(res.LastInsertId())
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
func (node *DBExecNode) GetDatasets(create bool) ([]*DBDataset, bool) {
	// get the old exec-dataset references
	// we'll need to update datasets that are no longer referenced
	existingDS := node.DatasetRefs()

	// find datasets that match current hash
	nodeHash := node.Hash()
	datasets := make([]*DBDataset, len(node.DataTypes))
	dsIDSet := make(map[int]bool)
	ok := true
	for i := range datasets {
		dsName := fmt.Sprintf("%s[%d]", node.Name, i)
		curHash := fmt.Sprintf("%s[%d]", nodeHash, i)
		ds := FindDataset(curHash)
		if ds == nil {
			ok = false
			if create {
				ds = NewDataset(dsName, "computed", node.DataTypes[i], &curHash)
			}
		}

		if ds != nil {
			ds.AddExecRef(node.ID)
			datasets[i] = ds
			dsIDSet[ds.ID] = true
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

type ExecNodeUpdate struct {
	Name *string
	Op *string
	Params *string
	Parents *[]skyhook.ExecParent
	DataTypes *[]skyhook.DataType
}

func (node *DBExecNode) Update(req ExecNodeUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE exec_nodes SET name = ? WHERE id = ?", *req.Name, node.ID)
	}
	if req.Op != nil {
		db.Exec("UPDATE exec_nodes SET op = ? WHERE id = ?", *req.Op, node.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE exec_nodes SET params = ? WHERE id = ?", *req.Params, node.ID)
	}
	if req.Parents != nil {
		db.Exec("UPDATE exec_nodes SET parents = ? WHERE id = ?", skyhook.ExecParentsToString(*req.Parents), node.ID)
	}
	if req.DataTypes != nil {
		var typesStr []string
		for _, t := range *req.DataTypes {
			typesStr = append(typesStr, string(t))
		}
		typesRaw := strings.Join(typesStr, ",")
		db.Exec("UPDATE exec_nodes SET data_types = ? WHERE id = ?", typesRaw, node.ID)
	}
}

func (node *DBExecNode) Delete() {
	dsIDs := node.DatasetRefs()
	for _, id := range dsIDs {
		GetDataset(id).DeleteExecRef(node.ID)
	}

	// TODO: check for other exec nodes that reference this node as a parent

	db.Exec("DELETE FROM exec_nodes WHERE id = ?", node.ID)
}
