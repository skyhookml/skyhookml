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
	sdb, err := sql.Open("sqlite3", "data/skyhook.sqlite3")
	if err != nil {
		panic(err)
	}
	db = &Database{db: sdb}
}

func (this *Database) Query(q string, args ...interface{}) *Rows {
	this.mu.Lock()
	if DbDebug {
		log.Printf("[db] Query: %v", q)
	}
	rows, err := this.db.Query(q, args...)
	if err != nil {
		this.mu.Unlock()
		panic(err)
	}
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
	if err != nil {
		panic(err)
	}
	return Result{result}
}

func (this *Database) Transaction(f func(tx Tx)) {
	this.mu.Lock()
	defer this.mu.Unlock()
	f(Tx{this})
}

func (this *Database) Close() {
	err := this.db.Close()
	if err != nil {
		panic(err)
	}
}

type Rows struct {
	db     *Database
	locked bool
	rows   *sql.Rows
}

func (r *Rows) Close() {
	err := r.rows.Close()
	if r.locked {
		r.db.mu.Unlock()
		r.locked = false
	}
	if err != nil {
		panic(err)
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
	if err != nil {
		if r.locked {
			r.db.mu.Unlock()
			r.locked = false
		}
		panic(err)
	}
}

type Row struct {
	db     *Database
	locked bool
	row    *sql.Row
}

func (r Row) Scan(dest ...interface{}) {
	err := r.row.Scan(dest...)
	if r.locked {
		r.db.mu.Unlock()
		r.locked = false
	}
	if err != nil {
		panic(err)
	}
}

type Result struct {
	result sql.Result
}

func (r Result) LastInsertId() int {
	id, err := r.result.LastInsertId()
	if err != nil {
		panic(err)
	}
	return int(id)
}

func (r Result) RowsAffected() int {
	count, err := r.result.RowsAffected()
	if err != nil {
		panic(err)
	}
	return int(count)
}

type Tx struct {
	db *Database
}

func (tx Tx) Query(q string, args ...interface{}) Rows {
	rows, err := tx.db.db.Query(q, args...)
	if err != nil {
		panic(err)
	}
	return Rows{tx.db, false, rows}
}

func (tx Tx) QueryRow(q string, args ...interface{}) Row {
	row := tx.db.db.QueryRow(q, args...)
	return Row{tx.db, false, row}
}

func (tx Tx) Exec(q string, args ...interface{}) Result {
	result, err := tx.db.db.Exec(q, args...)
	if err != nil {
		panic(err)
	}
	return Result{result}
}
