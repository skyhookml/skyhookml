package app

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
)

// Database cache for per-dataset sqlite3 files.

var dbCache = make(map[string]*Database)
var dbCacheMu sync.Mutex

func GetCachedDB(fname string) *Database {
	dbCacheMu.Lock()
	defer dbCacheMu.Unlock()
	if dbCache[fname] == nil {
		os.Mkdir(filepath.Dir(fname), 0755)
		sdb, err := sql.Open("sqlite3", fname)
		if err != nil {
			panic(err)
		}
		db := &Database{db: sdb}
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
		dbCache[fname] = db
	}
	return dbCache[fname]
}

func UncacheDB(fname string) {
	dbCacheMu.Lock()
	delete(dbCache, fname)
	dbCacheMu.Unlock()
}
