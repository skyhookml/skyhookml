package app

import (
	"github.com/skyhookml/skyhookml/skyhook"
)

type DBPytorchComponent struct {skyhook.PytorchComponent}
type DBPytorchArch struct {skyhook.PytorchArch}

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
