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
		id := r.PostForm.Get("id")
		comp := NewPytorchComponent(id)
		skyhook.JsonResponse(w, comp)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/components/{comp}", func(w http.ResponseWriter, r *http.Request) {
		comp := GetPytorchComponent(mux.Vars(r)["comp"])
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}
		skyhook.JsonResponse(w, comp)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/components/{comp}", func(w http.ResponseWriter, r *http.Request) {
		comp := GetPytorchComponent(mux.Vars(r)["comp"])
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

	Router.HandleFunc("/pytorch/components/{comp}", func(w http.ResponseWriter, r *http.Request) {
		comp := GetPytorchComponent(mux.Vars(r)["comp"])
		if comp == nil {
			http.Error(w, "no such PytorchComponent", 404)
			return
		}
		comp.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListPytorchArchs())
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := r.PostForm.Get("id")
		arch := NewPytorchArch(id)
		skyhook.JsonResponse(w, arch)
	}).Methods("POST")

	Router.HandleFunc("/pytorch/archs/{arch}", func(w http.ResponseWriter, r *http.Request) {
		arch := GetPytorchArchByName(mux.Vars(r)["arch"])
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}
		skyhook.JsonResponse(w, arch)
	}).Methods("GET")

	Router.HandleFunc("/pytorch/archs/{arch}", func(w http.ResponseWriter, r *http.Request) {
		arch := GetPytorchArchByName(mux.Vars(r)["arch"])
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

	Router.HandleFunc("/pytorch/archs/{arch}", func(w http.ResponseWriter, r *http.Request) {
		arch := GetPytorchArchByName(mux.Vars(r)["arch"])
		if arch == nil {
			http.Error(w, "no such PytorchArch", 404)
			return
		}
		arch.Delete()
	}).Methods("DELETE")
}
