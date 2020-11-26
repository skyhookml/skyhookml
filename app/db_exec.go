package app

import (
	"../skyhook"

	"fmt"
	"strconv"
	"strings"
)

type DBExecNode struct {skyhook.ExecNode}

const ExecNodeQuery = "SELECT id, name, op, params, parents, data_types, datasets FROM exec_nodes"

func execNodeListHelper(rows *Rows) []*DBExecNode {
	nodes := []*DBExecNode{}
	for rows.Next() {
		var node DBExecNode
		var parentsRaw, typesRaw, datasetsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsRaw, &typesRaw, &datasetsRaw)
		node.Parents = skyhook.ParseExecParents(parentsRaw)
		node.DataTypes = skyhook.DecodeTypes(typesRaw)

		node.DatasetIDs = make([]*int, len(node.DataTypes))
		for i, part := range strings.Split(datasetsRaw, ",") {
			if part == "" {
				continue
			}
			id := skyhook.ParseInt(part)
			node.DatasetIDs[i] = &id
		}
		nodes = append(nodes, &node)
	}
	return nodes
}

func ListExecNodes() []*DBExecNode {
	rows := db.Query(ExecNodeQuery)
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

func NewExecNode(name string, op string, params string, parents []skyhook.ExecParent, dataTypes []skyhook.DataType) *DBExecNode {
	var parentsStr []string
	for _, parent := range parents {
		parentsStr = append(parentsStr, parent.String())
	}
	res := db.Exec(
		"INSERT INTO exec_nodes (name, op, params, parents, data_types, datasets) VALUES (?, ?, ?, ?, ?, '')",
		name, op, params, strings.Join(parentsStr, ";"), skyhook.EncodeTypes(dataTypes),
	)
	return GetExecNode(res.LastInsertId())
}

// Create datasets for each output of this node.
func (node *DBExecNode) EnsureDatasets() {
	for i := range node.DatasetIDs {
		if node.DatasetIDs[i] != nil {
			continue
		}
		dsName := fmt.Sprintf("%s[%d]", node.Name, i)
		ds := NewDataset(dsName, "computed", node.DataTypes[i])
		id := ds.ID
		node.DatasetIDs[i] = &id
	}
	var idsStr []string
	for _, id := range node.DatasetIDs {
		idsStr = append(idsStr, strconv.Itoa(*id))
	}
	db.Exec("UPDATE exec_nodes SET datasets = ? WHERE id = ?", strings.Join(idsStr, ","), node.ID)
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
		var parentsStr []string
		for _, parent := range *req.Parents {
			parentsStr = append(parentsStr, parent.String())
		}
		parentsRaw := strings.Join(parentsStr, ";")
		db.Exec("UPDATE exec_nodes SET parents = ? WHERE id = ?", parentsRaw, node.ID)
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
