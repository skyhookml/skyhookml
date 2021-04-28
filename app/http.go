package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"net/http"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
)

var SetupFuncs []func(*socketio.Server)
var Router = mux.NewRouter()

func init() {
	fileServer := http.FileServer(http.Dir("web/dist/"))
	Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))
	Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})

	Router.HandleFunc("/data-types", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, skyhook.DataTypes)
	}).Methods("GET")

	Router.HandleFunc("/ops", func(w http.ResponseWriter, r *http.Request) {
		type Op struct {
			skyhook.ExecOpConfig
			Inputs []skyhook.ExecInput
			Outputs []skyhook.ExecOutput
		}
		ops := make(map[string]Op)
		for _, provider := range skyhook.ExecOpProviders {
			cfg := provider.Config()
			inputs := provider.GetInputs("")
			outputs := provider.GetOutputs("", nil)
			ops[cfg.ID] = Op{
				ExecOpConfig: cfg,
				Inputs: inputs,
				Outputs: outputs,
			}
		}
		skyhook.JsonResponse(w, ops)
	}).Methods("GET")
}
