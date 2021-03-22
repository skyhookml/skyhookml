package app

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Initialize the database on startup with cleanup operations.
// If init is true, we also first initialize the schema and populate certain tables.
func InitDB(init bool) {
	if init {
		db.Exec(`CREATE TABLE IF NOT EXISTS kv (
			k TEXT PRIMARY KEY,
			v TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS datasets (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			-- 'data' or 'computed'
			type TEXT,
			data_type TEXT,
			-- only set if computed
			hash TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS annotate_datasets (
			id INTEGER PRIMARY KEY ASC,
			dataset_id INTEGER REFERENCES datasets(id),
			inputs TEXT,
			tool TEXT,
			params TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS pytorch_archs (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			params TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS pytorch_components (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			params TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS exec_nodes (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			op TEXT,
			params TEXT,
			inputs TEXT,
			outputs TEXT,
			parents TEXT,
			workspace TEXT
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS exec_ds_refs (
			node_id INTEGER,
			dataset_id INTEGER,
			UNIQUE(node_id, dataset_id)
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS workspaces (
			name TEXT PRIMARY KEY
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS ws_datasets (
			dataset_id INTEGER,
			workspace TEXT,
			UNIQUE(dataset_id, workspace)
		)`)
		db.Exec(`CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY ASC,
			name TEXT,
			-- e.g. 'execnode'
			type TEXT,
			-- how to process the job output and render the job
			op TEXT,
			metadata TEXT,
			start_time TIMESTAMP,
			state TEXT DEFAULT '',
			done INTEGER DEFAULT 0,
			error TEXT DEFAULT ''
		)`)

		// add missing pytorch components
		componentPath := "python/skyhook/pytorch/components/"
		files, err := ioutil.ReadDir(componentPath)
		if err != nil {
			panic(err)
		}
		for _, fi := range files {
			if !strings.HasSuffix(fi.Name(), ".json") {
				continue
			}
			name := strings.Split(fi.Name(), ".json")[0]
			bytes, err := ioutil.ReadFile(filepath.Join(componentPath, fi.Name()))
			if err != nil {
				panic(err)
			}
			var count int
			db.QueryRow("SELECT COUNT(*) FROM pytorch_components WHERE name = ?", name).Scan(&count)
			if count == 0 {
				db.Exec("INSERT INTO pytorch_components (name, params) VALUES (?, ?)", name, string(bytes))
			} else {
				db.Exec("UPDATE pytorch_components SET params = ? WHERE name = ?", string(bytes), name)
			}
		}
	}

	// now run some database cleanup steps

	// mark jobs that are still running as error
	db.Exec("UPDATE jobs SET error = 'terminated' WHERE done = 0")

	// delete temporary datasetsTODO
}
