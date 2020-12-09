package app

import (
	"../skyhook"

	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func (item *DBItem) Handle(format string, w http.ResponseWriter, r *http.Request) {
	item.Load()

	if format == item.Format {
		http.ServeFile(w, r, item.Fname())
		return
	}

	file, err := os.Open(item.Fname())
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := skyhook.DecodeData(item.Dataset.DataType, item.Format, item.Metadata, file)
	if err != nil {
		panic(err)
	}

	if format == "jpeg" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else if format == "mp4" {
		w.Header().Set("Content-Type", "video/mp4")
	} else if format == "json" {
		w.Header().Set("Content-Type", "application/json")
	}
	if err := data.Encode(format, w); err != nil {
		panic(err)
	}
}

func init() {
	Router.HandleFunc("/datasets", func(w http.ResponseWriter, r *http.Request) {
		skyhook.JsonResponse(w, ListDatasets())
	}).Methods("GET")

	Router.HandleFunc("/datasets", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.PostForm.Get("name")
		dataType := r.PostForm.Get("data_type")
		ds := NewDataset(name, "data", skyhook.DataType(dataType))
		skyhook.JsonResponse(w, ds)
	}).Methods("POST")

	Router.HandleFunc("/datasets/{ds_id}", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		skyhook.JsonResponse(w, dataset)
	}).Methods("GET")

	Router.HandleFunc("/datasets/{ds_id}", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		dataset.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/datasets/{ds_id}/items", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		skyhook.JsonResponse(w, dataset.ListItems())
	}).Methods("GET")

	Router.HandleFunc("/items/{item_id}", func(w http.ResponseWriter, r *http.Request) {
		itemID := skyhook.ParseInt(mux.Vars(r)["item_id"])
		item := GetItem(itemID)
		if item == nil {
			http.Error(w, "no such item", 404)
			return
		}
		item.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/items/{item_id}/get", func(w http.ResponseWriter, r *http.Request) {
		itemID := skyhook.ParseInt(mux.Vars(r)["item_id"])
		item := GetItem(itemID)
		if item == nil {
			http.Error(w, "no such item", 404)
			return
		}
		r.ParseForm()
		format := r.Form.Get("format")
		item.Handle(format, w, r)
	}).Methods("GET")

	Router.HandleFunc("/datasets/{ds_id}/get-item", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		r.ParseForm()
		key := r.Form.Get("key")
		format := r.Form.Get("format")

		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		item := dataset.GetItem(key)
		if item == nil {
			http.Error(w, "no matching item", 404)
			return
		}

		if format == "meta" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(item.Metadata))
			return
		}

		item.Handle(format, w, r)
	})
}
