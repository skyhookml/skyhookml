package app

import (
	_ "github.com/mattn/go-sqlite3"

	"database/sql"
	"log"

	// use deadlock detector mutexes here since deadlocks in database operations
	// will be common
	sync "github.com/sasha-s/go-deadlock"
)

const DbDebug bool = false

var db *Database

type Database struct {
	db *sql.DB
	mu sync.Mutex
}

func GetDB() *Database {
	return db
}

func init() {
	sdb, err := sql.Open("sqlite3", "./skyhook.sqlite3")
	if err != nil {
		panic(err)
	}
	db = &Database{db: sdb}

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
	db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY ASC,
		dataset_id INTEGER REFERENCES datasets(id),
		-- item key
		k TEXT,
		ext TEXT,
		format TEXT,
		metadata TEXT
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
	db.Exec(`CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY ASC,
		hash TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS train_nodes (
		id INTEGER PRIMARY KEY ASC,
		name TEXT,
		op TEXT,
		params TEXT,
		parents TEXT,
		outputs TEXT,
		workspace TEXT,
		model_id INTEGER
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS exec_nodes (
		id INTEGER PRIMARY KEY ASC,
		name TEXT,
		op TEXT,
		params TEXT,
		parents TEXT,
		data_types TEXT,
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
}

func (this *Database) Query(q string, args ...interface{}) *Rows {
	this.mu.Lock()
	if DbDebug {
		log.Printf("[db] Query: %v", q)
	}
	rows, err := this.db.Query(q, args...)
	checkErr(err)
	return &Rows{this, true, rows}
}

func (this *Database) QueryRow(q string, args ...interface{}) *Row {
	this.mu.Lock()
	if DbDebug {
		log.Printf("[db] QueryRow: %v", q)
	}
	row := this.db.QueryRow(q, args...)
	return &Row{this, true, row}
}

func (this *Database) Exec(q string, args ...interface{}) Result {
	this.mu.Lock()
	defer this.mu.Unlock()
	if DbDebug {
		log.Printf("[db] Exec: %v", q)
	}
	result, err := this.db.Exec(q, args...)
	checkErr(err)
	return Result{result}
}

func (this *Database) Transaction(f func(tx Tx)) {
	this.mu.Lock()
	f(Tx{this})
	this.mu.Unlock()
}

type Rows struct {
	db     *Database
	locked bool
	rows   *sql.Rows
}

func (r *Rows) Close() {
	err := r.rows.Close()
	checkErr(err)
	if r.locked {
		r.db.mu.Unlock()
		r.locked = false
	}
}

func (r *Rows) Next() bool {
	hasNext := r.rows.Next()
	if !hasNext && r.locked {
		r.db.mu.Unlock()
		r.locked = false
	}
	return hasNext
}

func (r *Rows) Scan(dest ...interface{}) {
	err := r.rows.Scan(dest...)
	checkErr(err)
}

type Row struct {
	db     *Database
	locked bool
	row    *sql.Row
}

func (r Row) Scan(dest ...interface{}) {
	err := r.row.Scan(dest...)
	checkErr(err)
	if r.locked {
		r.db.mu.Unlock()
		r.locked = false
	}
}

type Result struct {
	result sql.Result
}

func (r Result) LastInsertId() int {
	id, err := r.result.LastInsertId()
	checkErr(err)
	return int(id)
}

func (r Result) RowsAffected() int {
	count, err := r.result.RowsAffected()
	checkErr(err)
	return int(count)
}

type Tx struct {
	db *Database
}

func (tx Tx) Query(q string, args ...interface{}) Rows {
	rows, err := tx.db.db.Query(q, args...)
	checkErr(err)
	return Rows{tx.db, false, rows}
}

func (tx Tx) QueryRow(q string, args ...interface{}) Row {
	row := tx.db.db.QueryRow(q, args...)
	return Row{tx.db, false, row}
}

func (tx Tx) Exec(q string, args ...interface{}) Result {
	result, err := tx.db.db.Exec(q, args...)
	checkErr(err)
	return Result{result}
}
