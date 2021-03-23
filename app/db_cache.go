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

// Get a cached database connection to the specified sqlite3 file.
// If initFunc is set, we create the sqlite3 if it doesn't already exist, and
//   call initFunc each time a new connection is opened.
// Otherwise, we do not create new files, and instead return nil.
func GetCachedDB(fname string, initFunc func(*Database)) *Database {
	dbCacheMu.Lock()
	defer dbCacheMu.Unlock()
	if dbCache[fname] == nil {
		var db *Database
		if initFunc == nil {
			if _, err := os.Stat(fname); os.IsNotExist(err) {
				return nil
			}
			sdb, err := sql.Open("sqlite3", fname)
			if err != nil {
				panic(err)
			}
			db = &Database{db: sdb}
		} else {
			os.Mkdir(filepath.Dir(fname), 0755)
			sdb, err := sql.Open("sqlite3", fname)
			if err != nil {
				panic(err)
			}
			db = &Database{db: sdb}
			initFunc(db)
		}
		dbCache[fname] = db
	}
	return dbCache[fname]
}

func UncacheDB(fname string) {
	dbCacheMu.Lock()
	db := dbCache[fname]
	if db != nil {
		delete(dbCache, fname)
		dbCacheMu.Unlock()
		db.Close()
		return
	}
	dbCacheMu.Unlock()
}
