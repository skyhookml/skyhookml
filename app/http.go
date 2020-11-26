package app

import (
	"net/http"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
)

var SetupFuncs []func(*socketio.Server)
var Router = mux.NewRouter()

func init() {
	fileServer := http.FileServer(http.Dir("static/"))
	Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))
	Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}
