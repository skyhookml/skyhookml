package skyhook

import (
	"fmt"
	"os"
)

type Dataset struct {
	ID int
	Name string

	// data or computed
	Type string

	DataType DataType
}

type Item struct {
	ID int
	Dataset Dataset
	Key string
	Ext string
	Format string
	Metadata string
}

func (item Item) Fname() string {
	return fmt.Sprintf("items/%d/%d.%s", item.Dataset.ID, item.ID, item.Ext)
}

func (item Item) Mkdir() {
	os.Mkdir(fmt.Sprintf("items/%d", item.Dataset.ID), 0755)
}

func (item Item) UpdateData(data Data) {
	item.Mkdir()
	file, err := os.Create(item.Fname())
	if err != nil {
		panic(err)
	}
	if err := data.Encode(item.Format, file); err != nil {
		panic(err)
	}
}

func (item Item) LoadData() (Data, error) {
	return DecodeFile(item.Dataset.DataType, item.Format, item.Metadata, item.Fname())
}

func (ds Dataset) Remove() {
	os.RemoveAll(fmt.Sprintf("items/%d", ds.ID))
}

func (item Item) Remove() {
	os.Remove(item.Fname())
}
