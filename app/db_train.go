package app

import (
	"../skyhook"

	"strconv"
	"strings"
)

type DBTrainNode struct {
	skyhook.TrainNode
	Workspace string
	ModelID *int
}
type DBPytorchComponent struct {skyhook.PytorchComponent}
type DBPytorchArch struct {skyhook.PytorchArch}
type DBModel struct {skyhook.Model}

const TrainNodeQuery = "SELECT id, name, op, params, parents, outputs, workspace, model_id FROM train_nodes"

func trainNodeListHelper(rows *Rows) []*DBTrainNode {
	nodes := []*DBTrainNode{}
	for rows.Next() {
		var node DBTrainNode
		var parentsStr, outputsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsStr, &outputsRaw, &node.Workspace, &node.ModelID)
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
	res := db.Exec("INSERT INTO train_nodes (name, op, params, parents, outputs, workspace) VALUES (?, ?, '', '', '', ?)", name, op, workspace)
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

func (node *DBTrainNode) Delete() {
	if node.ModelID != nil {
		model := GetModel(*node.ModelID)
		node.ModelID = nil
		db.Exec("UPDATE train_nodes SET model_id = NULL WHERE id = ?", node.ID)
		model.CheckRefs()
	}

	// TODO: check for other train nodes that reference this node as a parent

	db.Exec("DELETE FROM train_nodes WHERE id = ?", node.ID)
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

const ModelQuery = "SELECT id, hash FROM models"

func modelListHelper(rows *Rows) []*DBModel {
	models := []*DBModel{}
	for rows.Next() {
		var m DBModel
		rows.Scan(&m.ID, &m.Hash)
		models = append(models, &m)
	}
	return models
}

func ListModels() []*DBModel {
	rows := db.Query(ModelQuery)
	return modelListHelper(rows)
}

func GetModel(id int) *DBModel {
	rows := db.Query(ModelQuery + " WHERE id = ?", id)
	models := modelListHelper(rows)
	if len(models) == 1 {
		return models[0]
	} else {
		return nil
	}
}

func FindModel(hash string) *DBModel {
	rows := db.Query(ModelQuery + " WHERE hash = ?", hash)
	models := modelListHelper(rows)
	if len(models) == 1 {
		return models[0]
	} else {
		return nil
	}
}

// Delete if no more refs. Called after a TrainNode updates its model_id.
func (m *DBModel) CheckRefs() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM train_nodes WHERE model_id = ?", m.ID).Scan(&count)
	if count > 0 {
		return
	}
	m.Delete()
}

func (m *DBModel) Delete() {
	// TODO: delete from disk
	db.Exec("DELETE FROM models WHERE id = ?", m.ID)
}
