package app

import (
	"../skyhook"

	"strconv"
	"strings"
)

type DBTrainNode struct {
	skyhook.TrainNode
	Workspace string
}
type DBPytorchComponent struct {skyhook.PytorchComponent}
type DBPytorchArch struct {skyhook.PytorchArch}

const TrainNodeQuery = "SELECT id, name, op, params, parents, outputs, trained, workspace FROM train_nodes"

func trainNodeListHelper(rows *Rows) []*DBTrainNode {
	nodes := []*DBTrainNode{}
	for rows.Next() {
		var node DBTrainNode
		var parentsStr, outputsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsStr, &outputsRaw, &node.Trained, &node.Workspace)
		for _, s := range strings.Split(parentsStr, ",") {
			if s == "" {
				continue
			}
			node.ParentIDs = append(node.ParentIDs, skyhook.ParseInt(s))
		}
		node.Outputs = skyhook.DecodeTypes(outputsRaw)
		nodes = append(nodes, &node)
	}
	return nodes
}

func ListTrainNodes() []*DBTrainNode {
	rows := db.Query(TrainNodeQuery)
	return trainNodeListHelper(rows)
}

func (ws DBWorkspace) ListTrainNodes() []*DBTrainNode {
	rows := db.Query(TrainNodeQuery + " WHERE workspace = ?", ws)
	return trainNodeListHelper(rows)
}

func GetTrainNode(id int) *DBTrainNode {
	rows := db.Query(TrainNodeQuery + " WHERE id = ?", id)
	nodes := trainNodeListHelper(rows)
	if len(nodes) == 1 {
		return nodes[0]
	} else {
		return nil
	}
}

func NewTrainNode(name string, op string, workspace string) *DBTrainNode {
	res := db.Exec("INSERT INTO train_nodes (name, op, params, parents, outputs, trained, workspace) VALUES (?, ?, '', '', '', 0, ?)", name, op, workspace)
	return GetTrainNode(res.LastInsertId())
}

type TrainNodeUpdate struct {
	Name *string
	Op *string
	Params *string
	ParentIDs *[]int
	Outputs *[]skyhook.DataType
}

func (node *DBTrainNode) Update(req TrainNodeUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE train_nodes SET name = ? WHERE id = ?", *req.Name, node.ID)
	}
	if req.Op != nil {
		db.Exec("UPDATE train_nodes SET op = ? WHERE id = ?", *req.Op, node.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE train_nodes SET params = ? WHERE id = ?", *req.Params, node.ID)
	}
	if req.ParentIDs != nil {
		var strs []string
		for _, id := range *req.ParentIDs {
			strs = append(strs, strconv.Itoa(id))
		}
		db.Exec("UPDATE train_nodes SET parents = ? WHERE id = ?", strings.Join(strs, ","), node.ID)
	}
	if req.Outputs != nil {
		db.Exec("UPDATE train_nodes SET outputs = ? WHERE id = ?", skyhook.EncodeTypes(*req.Outputs), node.ID)
	}
}

const PytorchComponentQuery = "SELECT id, name, params FROM pytorch_components"

func pytorchComponentListHelper(rows *Rows) []*DBPytorchComponent {
	var comps []*DBPytorchComponent
	for rows.Next() {
		var c DBPytorchComponent
		var paramsRaw string
		rows.Scan(&c.ID, &c.Name, &paramsRaw)
		skyhook.JsonUnmarshal([]byte(paramsRaw), &c.Params)
		comps = append(comps, &c)
	}
	return comps
}

func ListPytorchComponents() []*DBPytorchComponent {
	rows := db.Query(PytorchComponentQuery)
	return pytorchComponentListHelper(rows)
}

func GetPytorchComponent(id int) *DBPytorchComponent {
	rows := db.Query(PytorchComponentQuery + " WHERE id = ?", id)
	comps := pytorchComponentListHelper(rows)
	if len(comps) == 1 {
		return comps[0]
	} else {
		return nil
	}
}

func NewPytorchComponent(name string) *DBPytorchComponent {
	res := db.Exec("INSERT INTO pytorch_components (name, params) VALUES (?, '{}')", name)
	return GetPytorchComponent(res.LastInsertId())
}

type PytorchComponentUpdate struct {
	Name *string
	Params *skyhook.PytorchComponentParams
}

func (c *DBPytorchComponent) Update(req PytorchComponentUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE pytorch_components SET name = ? WHERE id = ?", *req.Name, c.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE pytorch_components SET params = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Params)), c.ID)
	}
}

const PytorchArchQuery = "SELECT id, name, params FROM pytorch_archs"

func pytorchArchListHelper(rows *Rows) []*DBPytorchArch {
	var archs []*DBPytorchArch
	for rows.Next() {
		var arch DBPytorchArch
		var paramsRaw string
		rows.Scan(&arch.ID, &arch.Name, &paramsRaw)
		skyhook.JsonUnmarshal([]byte(paramsRaw), &arch.Params)
		archs = append(archs, &arch)
	}
	return archs
}

func ListPytorchArchs() []*DBPytorchArch {
	rows := db.Query(PytorchArchQuery)
	return pytorchArchListHelper(rows)
}

func GetPytorchArch(id int) *DBPytorchArch {
	rows := db.Query(PytorchArchQuery + " WHERE id = ?", id)
	archs := pytorchArchListHelper(rows)
	if len(archs) == 1 {
		return archs[0]
	} else {
		return nil
	}
}

func NewPytorchArch(name string) *DBPytorchArch {
	res := db.Exec("INSERT INTO pytorch_archs (name, params) VALUES (?, '{}')", name)
	return GetPytorchArch(res.LastInsertId())
}

type PytorchArchUpdate struct {
	Name *string
	Params *skyhook.PytorchArchParams
}

func (arch *DBPytorchArch) Update(req PytorchArchUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE pytorch_archs SET name = ? WHERE id = ?", *req.Name, arch.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE pytorch_archs SET params = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Params)), arch.ID)
	}
}
