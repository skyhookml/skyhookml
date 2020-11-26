package app

import (
	"../skyhook"

	"math/rand"
	"strings"
)

type DBDataset struct {skyhook.Dataset}
type DBAnnotateDataset struct {
	skyhook.AnnotateDataset
	loaded bool
}
type DBItem struct {
	skyhook.Item
	loaded bool
}

const DatasetQuery = "SELECT id, name, type, data_type FROM datasets"

func datasetListHelper(rows *Rows) []*DBDataset {
	datasets := []*DBDataset{}
	for rows.Next() {
		var ds DBDataset
		rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.DataType)
		datasets = append(datasets, &ds)
	}
	return datasets
}

func ListDatasets() []*DBDataset {
	rows := db.Query(DatasetQuery)
	return datasetListHelper(rows)
}

func GetDataset(id int) *DBDataset {
	rows := db.Query(DatasetQuery + " WHERE id = ?", id)
	datasets := datasetListHelper(rows)
	if len(datasets) == 1 {
		return datasets[0]
	} else {
		return nil
	}
}

const AnnotateDatasetQuery = "SELECT a.id, d.id, d.name, d.type, d.data_type, a.inputs, a.tool, a.params FROM annotate_datasets AS a LEFT JOIN datasets AS d ON a.dataset_id = d.id"

func annotateDatasetListHelper(rows *Rows) []*DBAnnotateDataset {
	annosets := []*DBAnnotateDataset{}
	for rows.Next() {
		var s DBAnnotateDataset
		var inputsRaw string
		rows.Scan(&s.ID, &s.Dataset.ID, &s.Dataset.Name, &s.Dataset.Type, &s.Dataset.DataType, &inputsRaw, &s.Tool, &s.Params)
		for _, part := range strings.Split(inputsRaw, ",") {
			s.Inputs = append(s.Inputs, skyhook.Dataset{
				ID: skyhook.ParseInt(part),
			})
		}
		annosets = append(annosets, &s)
	}
	return annosets
}

func ListAnnotateDatasets() []*DBAnnotateDataset {
	rows := db.Query(AnnotateDatasetQuery)
	return annotateDatasetListHelper(rows)
}

func GetAnnotateDataset(id int) *DBAnnotateDataset {
	rows := db.Query(AnnotateDatasetQuery + " WHERE a.id = ?", id)
	annosets := annotateDatasetListHelper(rows)
	if len(annosets) == 1 {
		return annosets[0]
	} else {
		return nil
	}
}

func (s *DBAnnotateDataset) Load() {
	if s.loaded {
		return
	}

	s.Dataset = GetDataset(s.Dataset.ID).Dataset
	for i := range s.Inputs {
		s.Inputs[i] = GetDataset(s.Inputs[i].ID).Dataset
	}
	s.loaded = true
}

// samples a key that is present in all input datasets but not yet labeled in this annotate dataset
// TODO: have sampler object so that hash tables can be stored in memory instead of loaded from db each time
func (s *DBAnnotateDataset) SampleMissingKey() string {
	var keys map[string]bool
	for _, ds := range s.Inputs {
		items := (&DBDataset{Dataset: ds}).ListItems()
		curKeys := make(map[string]bool)
		for _, item := range items {
			curKeys[item.Key] = true
		}
		if keys == nil {
			keys = curKeys
		} else {
			for key := range keys {
				if !curKeys[key] {
					delete(keys, key)
				}
			}
		}
	}

	items := (&DBDataset{Dataset: s.Dataset}).ListItems()
	for _, item := range items {
		delete(keys, item.Key)
	}

	var keyList []string
	for key := range keys {
		keyList = append(keyList, key)
	}
	if len(keyList) == 0 {
		return ""
	}
	return keyList[rand.Intn(len(keyList))]
}

const ItemQuery = "SELECT id, dataset_id, k, ext, format, metadata FROM items"

func itemListHelper(rows *Rows) []*DBItem {
	var items []*DBItem
	for rows.Next() {
		var item DBItem
		rows.Scan(&item.ID, &item.Dataset.ID, &item.Key, &item.Ext, &item.Format, &item.Metadata)
		items = append(items, &item)
	}
	return items
}

func (ds *DBDataset) ListItems() []*DBItem {
	rows := db.Query(ItemQuery + " WHERE dataset_id = ? ORDER BY id", ds.ID)
	items := itemListHelper(rows)
	// populate dataset
	for _, item := range items {
		item.Dataset = ds.Dataset
		item.loaded = true
	}
	return items
}

func GetItem(id int) *DBItem {
	rows := db.Query(ItemQuery + " WHERE id = ?", id)
	items := itemListHelper(rows)
	if len(items) == 1 {
		return items[0]
	} else {
		return nil
	}
}

func (ds *DBDataset) AddItem(key string, ext string, format string, metadata string) *DBItem {
	res := db.Exec(
		"INSERT INTO items (dataset_id, k, ext, format, metadata) VALUES (?, ?, ?, ?, ?)",
		ds.ID, key, ext, format, metadata,
	)
	return GetItem(res.LastInsertId())
}

func (ds *DBDataset) GetItem(key string) *DBItem {
	rows := db.Query(ItemQuery + " WHERE dataset_id = ? AND k = ? LIMIT 1", ds.ID, key)
	items := itemListHelper(rows)
	if len(items) == 1 {
		return items[0]
	} else {
		return nil
	}
}

func (ds *DBDataset) WriteItem(key string, data skyhook.Data) *DBItem {
	ext, format := data.GetDefaultExtAndFormat()
	item := ds.AddItem(key, ext, format, string(skyhook.JsonMarshal(data.GetMetadata())))
	item.UpdateData(data)
	return item
}

func (ds *DBDataset) Delete() {
	ds.Dataset.Remove()
	db.Exec("DELETE FROM items WHERE dataset_id = ?", ds.ID)
	db.Exec("DELETE FROM datasets")
}

func (item *DBItem) Delete() {
	item.Item.Remove()
	db.Exec("DELETE FROM items WHERE id = ?", item.ID)
}

func (item *DBItem) Load() {
	if item.loaded {
		return
	}
	item.Dataset = GetDataset(item.Dataset.ID).Dataset
	item.loaded = true
}

// Set metadata based on the file.
func (item *DBItem) SetMetadata() error {
	item.Load()
	format, metadata, err := skyhook.DataImpls[item.Dataset.DataType].GetDefaultMetadata(item.Fname())
	if err != nil {
		return err
	}
	item.Format = format
	item.Metadata = metadata
	db.Exec("UPDATE items SET format = ?, metadata = ? WHERE id = ?", item.Format, item.Metadata, item.ID)
	return nil
}

func NewDataset(name string, t string, dataType skyhook.DataType) *DBDataset {
	res := db.Exec("INSERT INTO datasets (name, type, data_type) VALUES (?, ?, ?)", name, t, dataType)
	return GetDataset(res.LastInsertId())
}
