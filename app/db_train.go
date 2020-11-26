package app

import (
	"../skyhook"

	"strconv"
	"strings"
)

type DBTrainNode struct {skyhook.TrainNode}
type DBKerasArch struct {skyhook.KerasArch}

const TrainNodeQuery = "SELECT id, name, op, params, parents, outputs, trained FROM train_nodes"

func trainNodeListHelper(rows *Rows) []*DBTrainNode {
	nodes := []*DBTrainNode{}
	for rows.Next() {
		var node DBTrainNode
		var parentsStr, outputsRaw string
		rows.Scan(&node.ID, &node.Name, &node.Op, &node.Params, &parentsStr, &outputsRaw, &node.Trained)
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

func GetTrainNode(id int) *DBTrainNode {
	rows := db.Query(TrainNodeQuery + " WHERE id = ?", id)
	nodes := trainNodeListHelper(rows)
	if len(nodes) == 1 {
		return nodes[0]
	} else {
		return nil
	}
}

func NewTrainNode(name string, op string) *DBTrainNode {
	res := db.Exec("INSERT INTO train_nodes (name, op, params, parents, outputs, trained) VALUES (?, ?, '', '', '', 0)", name, op)
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

/*const ModelQuery = "SELECT id, name, op, params, outputs FROM models"

func modelListHelper(rows *Rows) []*DBModel {
	models := []*DBModel{}
	for rows.Next() {
		var model DBModel
		var outputsRaw string
		rows.Scan(&model.ID, &model.Name, &model.Op, &model.Params, &outputsRaw)
		model.Outputs = make(map[string]skyhook.DataType)
		for _, s := range strings.Split(outputsRaw, ",") {
			parts := strings.Split(s, "=")
			model.Outputs[parts[0]] = skyhook.DataType(parts[1])
		}
		models = append(models, &model)
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
}*/

const KerasArchQuery = "SELECT id, name, params FROM keras_archs"

func kerasArchListHelper(rows *Rows) []*DBKerasArch {
	var archs []*DBKerasArch
	for rows.Next() {
		var arch DBKerasArch
		var paramsRaw string
		rows.Scan(&arch.ID, &arch.Name, &paramsRaw)
		skyhook.JsonUnmarshal([]byte(paramsRaw), &arch.Params)
		archs = append(archs, &arch)
	}
	return archs
}

func ListKerasArchs() []*DBKerasArch {
	rows := db.Query(KerasArchQuery)
	return kerasArchListHelper(rows)
}

func GetKerasArch(id int) *DBKerasArch {
	rows := db.Query(KerasArchQuery + " WHERE id = ?", id)
	archs := kerasArchListHelper(rows)
	if len(archs) == 1 {
		return archs[0]
	} else {
		return nil
	}
}

func NewKerasArch(name string) *DBKerasArch {
	res := db.Exec("INSERT INTO keras_archs (name, params) VALUES (?, '{}')", name)
	return GetKerasArch(res.LastInsertId())
}

type KerasArchUpdate struct {
	Name *string
	Params *skyhook.KerasArchParams
}

func (arch *DBKerasArch) Update(req KerasArchUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE keras_archs SET name = ? WHERE id = ?", *req.Name, arch.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE keras_archs SET params = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Params)), arch.ID)
	}
}
