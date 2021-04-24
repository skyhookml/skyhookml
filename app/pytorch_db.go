package app

import (
	"github.com/skyhookml/skyhookml/skyhook"
)

type DBPytorchComponent struct {skyhook.PytorchComponent}
type DBPytorchArch struct {skyhook.PytorchArch}

const PytorchComponentQuery = "SELECT id, params FROM pytorch_components"

func pytorchComponentListHelper(rows *Rows) []*DBPytorchComponent {
	var comps []*DBPytorchComponent
	for rows.Next() {
		var c DBPytorchComponent
		var paramsRaw string
		rows.Scan(&c.ID, &paramsRaw)
		skyhook.JsonUnmarshal([]byte(paramsRaw), &c.Params)
		comps = append(comps, &c)
	}
	return comps
}

func ListPytorchComponents() []*DBPytorchComponent {
	rows := db.Query(PytorchComponentQuery)
	return pytorchComponentListHelper(rows)
}

func GetPytorchComponent(id string) *DBPytorchComponent {
	rows := db.Query(PytorchComponentQuery + " WHERE id = ?", id)
	comps := pytorchComponentListHelper(rows)
	if len(comps) == 1 {
		return comps[0]
	} else {
		return nil
	}
}

func NewPytorchComponent(id string) *DBPytorchComponent {
	db.Exec("INSERT INTO pytorch_components (id, params) VALUES (?, '{}')", id)
	return GetPytorchComponent(id)
}

type PytorchComponentUpdate struct {
	Params *skyhook.PytorchComponentParams
}

func (c *DBPytorchComponent) Update(req PytorchComponentUpdate) {
	if req.Params != nil {
		db.Exec("UPDATE pytorch_components SET params = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Params)), c.ID)
	}
}

func (c *DBPytorchComponent) Delete() {
	db.Exec("DELETE FROM pytorch_components WHERE id = ?", c.ID)
}

const PytorchArchQuery = "SELECT id, params FROM pytorch_archs"

func pytorchArchListHelper(rows *Rows) []*DBPytorchArch {
	var archs []*DBPytorchArch
	for rows.Next() {
		var arch DBPytorchArch
		var paramsRaw string
		rows.Scan(&arch.ID, &paramsRaw)
		skyhook.JsonUnmarshal([]byte(paramsRaw), &arch.Params)
		archs = append(archs, &arch)
	}
	return archs
}

func ListPytorchArchs() []*DBPytorchArch {
	rows := db.Query(PytorchArchQuery)
	return pytorchArchListHelper(rows)
}

func GetPytorchArch(id string) *DBPytorchArch {
	rows := db.Query(PytorchArchQuery + " WHERE id = ?", id)
	archs := pytorchArchListHelper(rows)
	if len(archs) == 1 {
		return archs[0]
	} else {
		return nil
	}
}

func GetPytorchArchByName(id string) *DBPytorchArch {
	rows := db.Query(PytorchArchQuery + " WHERE id = ?", id)
	archs := pytorchArchListHelper(rows)
	if len(archs) == 1 {
		return archs[0]
	} else {
		return nil
	}
}

func NewPytorchArch(id string) *DBPytorchArch {
	db.Exec("INSERT INTO pytorch_archs (id, params) VALUES (?, '{}')", id)
	return GetPytorchArch(id)
}

type PytorchArchUpdate struct {
	Params *skyhook.PytorchArchParams
}

func (arch *DBPytorchArch) Update(req PytorchArchUpdate) {
	if req.Params != nil {
		db.Exec("UPDATE pytorch_archs SET params = ? WHERE id = ?", string(skyhook.JsonMarshal(*req.Params)), arch.ID)
	}
}

func (arch *DBPytorchArch) Delete() {
	db.Exec("DELETE FROM pytorch_archs WHERE id = ?", arch.ID)
}
