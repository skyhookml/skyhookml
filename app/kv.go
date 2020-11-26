package app

import (
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	Router.HandleFunc("/kv/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		var val string
		rows := db.Query("SELECT v FROM kv WHERE k = ?", key)
		if rows.Next() {
			rows.Scan(&val)
			rows.Close()
		}
		w.Write([]byte(val))
	}).Methods("GET")

	Router.HandleFunc("/kv/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		r.ParseForm()
		val := r.PostForm.Get("val")
		db.Transaction(func(tx Tx) {
			var count int
			tx.QueryRow("SELECT COUNT(*) FROM kv WHERE k = ?", key).Scan(&count)
			if count == 0 {
				tx.Exec("INSERT INTO kv (k, v) VALUES (?, ?)", key, val)
			} else {
				tx.Exec("UPDATE kv SET v = ? WHERE k = ?", val, key)
			}
		})
	}).Methods("POST")
}
