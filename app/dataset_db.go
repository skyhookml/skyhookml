package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
	"math/rand"
	"strings"
)

type DBDataset struct {
	skyhook.Dataset
	Done bool
}
type DBAnnotateDataset struct {
	skyhook.AnnotateDataset
	loaded bool
	InputDatasets []skyhook.Dataset
}
type DBItem struct {
	skyhook.Item
	loaded bool
}

const DatasetQuery = "SELECT id, name, type, data_type, metadata, hash, done FROM datasets"

func datasetListHelper(rows *Rows) []*DBDataset {
	datasets := []*DBDataset{}
	for rows.Next() {
		var ds DBDataset
		rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.DataType, &ds.Metadata, &ds.Hash, &ds.Done)
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

func FindDataset(hash string) *DBDataset {
	rows := db.Query(DatasetQuery + " WHERE hash = ?", hash)
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
		skyhook.JsonUnmarshal([]byte(inputsRaw), &s.Inputs)
		if s.Inputs == nil {
			s.Inputs = []skyhook.ExecParent{}
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
	s.InputDatasets = make([]skyhook.Dataset, len(s.Inputs))
	for i, input := range s.Inputs {
		ds, err := ExecParentToDataset(input)
		if err != nil {
			continue
		}
		s.InputDatasets[i] = ds.Dataset
	}
	s.loaded = true
}

// samples a key that is present in all input datasets but not yet labeled in this annotate dataset
// TODO: have sampler object so that hash tables can be stored in memory instead of loaded from db each time
func (s *DBAnnotateDataset) SampleMissingKey() string {
	var keys map[string]bool
	for _, parent := range s.Inputs {
		ds, err := ExecParentToDataset(parent)
		if err != nil {
			// TODO: probably want to handle this error somehow
			continue
		}
		items := ds.ListItems()
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

type AnnotateDatasetUpdate struct {
	Tool *string
	Params *string
}

func (s *DBAnnotateDataset) Update(req AnnotateDatasetUpdate) {
	if req.Tool != nil {
		db.Exec("UPDATE annotate_datasets SET tool = ? WHERE id = ?", *req.Tool, s.ID)
	}
	if req.Params != nil {
		db.Exec("UPDATE annotate_datasets SET params = ? WHERE id = ?", *req.Params, s.ID)
	}
}

func (s *DBAnnotateDataset) Delete() {
	db.Exec("DELETE FROM annotate_datasets WHERE id = ?", s.ID)
}

const ItemQuery = "SELECT k, ext, format, metadata, provider, provider_info FROM items"

func itemListHelper(rows *Rows) []*DBItem {
	var items []*DBItem
	for rows.Next() {
		var item DBItem
		rows.Scan(&item.Key, &item.Ext, &item.Format, &item.Metadata, &item.Provider, &item.ProviderInfo)
		items = append(items, &item)
	}
	return items
}

func (ds *DBDataset) getDB() *Database {
	return GetCachedDB(ds.DBFname(), func(db *Database) {
		db.Exec(`CREATE TABLE IF NOT EXISTS items (
			-- item key
			k TEXT PRIMARY KEY,
			ext TEXT,
			format TEXT,
			metadata TEXT,
			-- set if LoadData call should go through non-default method, else NULL
			provider TEXT,
			provider_info TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS datasets (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			-- 'data' or 'computed'
			type TEXT,
			data_type TEXT,
			metadata TEXT DEFAULT '',
			-- only set if computed
			hash TEXT
		)`)
		db.Exec(
			"INSERT OR REPLACE INTO datasets (id, name, type, data_type, metadata, hash) VALUES (1, ?, ?, ?, ?, ?)",
			ds.Name, ds.Type, ds.DataType, ds.Metadata, ds.Hash,
		)
	})
}

func (ds *DBDataset) ListItems() []*DBItem {
	db := ds.getDB()
	rows := db.Query(ItemQuery + " ORDER BY k")
	items := itemListHelper(rows)
	// populate dataset
	for _, item := range items {
		item.Dataset = ds.Dataset
		item.loaded = true
	}
	return items
}

func (ds *DBDataset) AddItem(item skyhook.Item) (*DBItem, error) {
	db := ds.getDB()
	// We use underlying Exec directly here since it is expected that we may encounter
	// a unique key constraint error.
	err := func() error {
		db.mu.Lock()
		defer db.mu.Unlock()
		_, err := db.db.Exec(
			"INSERT INTO items (k, ext, format, metadata, provider, provider_info) VALUES (?, ?, ?, ?, ?, ?)",
			item.Key, item.Ext, item.Format, item.Metadata, item.Provider, item.ProviderInfo,
		)
		return err
	}()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("item with key %s already exists in the dataset", item.Key)
		}
		return nil, err
	}
	return ds.GetItem(item.Key), nil
}

func (ds *DBDataset) GetItem(key string) *DBItem {
	db := ds.getDB()
	rows := db.Query(ItemQuery + " WHERE k = ?", key)
	items := itemListHelper(rows)
	if len(items) == 1 {
		item := items[0]
		item.Dataset = ds.Dataset
		item.loaded = true
		return item
	} else {
		return nil
	}
}

func (ds *DBDataset) WriteItem(key string, data interface{}, metadata skyhook.DataMetadata) (*DBItem, error) {
	// TODO: might want to write the item before updating database
	ext, format := ds.DataSpec().GetDefaultExtAndFormat(data, metadata)
	item, err := ds.AddItem(skyhook.Item{
		Key: key,
		Ext: ext,
		Format: format,
		Metadata: string(skyhook.JsonMarshal(metadata)),
	})
	if err != nil {
		return nil, err
	}
	err = item.UpdateData(data, metadata)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (ds *DBDataset) Delete() {
	ds.Clear()
	DeleteReferencesToDataset(ds)
	db.Exec("DELETE FROM datasets WHERE id = ?", ds.ID)
	db.Exec("DELETE FROM exec_ds_refs WHERE dataset_id = ?", ds.ID)
}

// Clear the dataset without deleting it.
func (ds *DBDataset) Clear() {
	ds.Dataset.Remove()
	UncacheDB(ds.DBFname())
}

func (ds *DBDataset) AddExecRef(nodeID int) {
	db.Exec("INSERT OR IGNORE INTO exec_ds_refs (node_id, dataset_id) VALUES (?, ?)", nodeID, ds.ID)
}

func (ds *DBDataset) DeleteExecRef(nodeID int) {
	db.Exec("DELETE FROM exec_ds_refs WHERE node_id = ? AND dataset_id = ?", nodeID, ds.ID)

	// if dataset no longer has any references, then we should delete the dataset
	var count int
	db.QueryRow("SELECT COUNT(*) from exec_ds_refs WHERE dataset_id = ?", ds.ID).Scan(&count)
	if count > 0 {
		return
	}
	log.Printf("[dataset %d-%s] removing empty dataset", ds.ID, ds.Name)
	ds.Delete()
}

func (ds *DBDataset) SetDone(done bool) {
	db.Exec("UPDATE datasets SET done = ? WHERE id = ?", done, ds.ID)
}

func (item *DBItem) Delete() {
	db := (&DBDataset{Dataset: item.Dataset}).getDB()
	db.Exec("DELETE FROM items WHERE k = ?", item.Key)
	item.Item.Remove()
}

func (item *DBItem) Load() {
	if item.loaded {
		return
	}
	item.Dataset = GetDataset(item.Dataset.ID).Dataset
	item.loaded = true
}

// Set metadata based on the file.
func (item *DBItem) SetMetadataFromFile() error {
	item.Load()
	fname := item.Fname()
	if fname == "" {
		return fmt.Errorf("could not set metadata from file in dataset not supporting filename")
	}
	spec := item.DataSpec()
	spec_, ok := spec.(skyhook.MetadataFromFileDataSpec)
	if !ok {
		return fmt.Errorf("MetadataFromFile not supported for type %s", item.Dataset.DataType)
	}
	format, metadata, err := spec_.GetMetadataFromFile(fname)
	if err != nil {
		return err
	}
	item.SetMetadata(format, metadata)
	return nil
}

func (item *DBItem) SetMetadata(format string, metadata skyhook.DataMetadata) {
	item.Load()
	item.Format = format
	item.Metadata = string(skyhook.JsonMarshal(metadata))
	db := (&DBDataset{Dataset: item.Dataset}).getDB()
	db.Exec("UPDATE items SET format = ?, metadata = ? WHERE k = ?", item.Format, item.Metadata, item.Key)
}

func NewDataset(name string, t string, dataType skyhook.DataType, hash *string) *DBDataset {
	done := t != "computed"
	res := db.Exec("INSERT INTO datasets (name, type, data_type, hash, done) VALUES (?, ?, ?, ?, ?)", name, t, dataType, hash, done)
	id := res.LastInsertId()
	log.Printf("[dataset %d-%s] created new dataset, data_type=%v", id, name, dataType)
	return GetDataset(id)
}

type DatasetUpdate struct {
	Name *string
	Metadata *string
}

func (ds *DBDataset) Update(req DatasetUpdate) {
	if req.Name != nil {
		db.Exec("UPDATE datasets SET name = ? WHERE id = ?", *req.Name, ds.ID)
		ds.Name = *req.Name
	}
	if req.Metadata != nil {
		db.Exec("UPDATE datasets SET metadata = ? WHERE id = ?", *req.Metadata, ds.ID)
		ds.Metadata = *req.Metadata
	}
}
