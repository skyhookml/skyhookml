package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	Router.HandleFunc("/pytorch/components", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListPytorchComponents())
	}).Methods("GET")

	Router.HandleFunc("/pytorch/components", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		comp := NewPytorchComponent(name)
		skyhook.JsonResponse(w, comp)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/components/{comp_id}", func(w http.ResponseWriter, r *http.Request) {
		compID := skyhook.ParseInt(mux.Vars(r)["comp_id"])
		comp := GetPytorchComponent(compID)
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}
		skyhook.JsonResponse(w, comp)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/components/{comp_id}", func(w http.ResponseWriter, r *http.Request) {
		compID := skyhook.ParseInt(mux.Vars(r)["comp_id"])
		comp := GetPytorchComponent(compID)
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}

		var request PytorchComponentUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		comp.Update(request)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListPytorchArchs())
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		arch := NewPytorchArch(name)
		skyhook.JsonResponse(w, arch)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/archs/{arch_id}", func(w http.ResponseWriter, r *http.Request) {
		archID := skyhook.ParseInt(mux.Vars(r)["arch_id"])
		arch := GetPytorchArch(archID)
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}
		skyhook.JsonResponse(w, arch)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs/{arch_id}", func(w http.ResponseWriter, r *http.Request) {
		archID := skyhook.ParseInt(mux.Vars(r)["arch_id"])
		arch := GetPytorchArch(archID)
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}

		var request PytorchArchUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		arch.Update(request)
	}).Methods("POST")
}
