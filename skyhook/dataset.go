package skyhook

import (
	"crypto/sha256"
	"fmt"
	"os"
)

type Dataset struct {
	ID int
	Name string

	// data or computed
	Type string

	DataType DataType

	// nil unless Type=computed
	Hash *string
}

type Item struct {
	Dataset Dataset
	Key string
	Ext string
	Format string
	Metadata string

	// nil to use default storage provider for LoadData / UpdateData
	Provider *string
	ProviderInfo *string
}

func (item Item) Fname() string {
	return fmt.Sprintf("items/%d/%s.%s", item.Dataset.ID, item.Key, item.Ext)
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
	if item.Provider == nil {
		return DefaultItemProvider(item)
	} else {
		return ItemProviders[*item.Provider](item)
	}
}

func (ds Dataset) Remove() {
	os.RemoveAll(fmt.Sprintf("items/%d", ds.ID))
}

func (item Item) Remove() {
	os.Remove(item.Fname())
}

func (ds Dataset) DBFname() string {
	return fmt.Sprintf("items/%d/db.sqlite3", ds.ID)
}

func (ds Dataset) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("name=%s\n", ds.Name)))
	return h.Sum(nil)
}

type ItemProvider func(item Item) (Data, error)
var ItemProviders = make(map[string]ItemProvider)

func DefaultItemProvider(item Item) (Data, error) {
	return DecodeFile(item.Dataset.DataType, item.Format, item.Metadata, item.Fname())
}
