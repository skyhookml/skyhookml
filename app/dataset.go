package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"log"
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
		ds := NewDataset(name, "data", skyhook.DataType(dataType), nil)
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

	Router.HandleFunc("/datasets/{ds_id}/items", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		r.ParseForm()
		key := r.Form.Get("key")
		ext := r.Form.Get("ext")
		format := r.Form.Get("format")
		metadata := r.Form.Get("metadata")
		provider := r.Form.Get("provider")
		providerInfo := r.Form.Get("provider_info")
		log.Printf("add item %s to dataset %d", key, dsID)

		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}

		item := skyhook.Item{
			Key: key,
			Ext: ext,
			Format: format,
			Metadata: metadata,
		}
		if provider != "" {
			item.Provider = new(string)
			*item.Provider = provider
			item.ProviderInfo = new(string)
			*item.ProviderInfo = providerInfo
		}

		item_ := dataset.AddItem(item)
		skyhook.JsonResponse(w, item_)
	}).Methods("POST")

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		itemKey := mux.Vars(r)["item_key"]

		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		item := dataset.GetItem(itemKey)
		if item == nil {
			http.Error(w, "no such item", 404)
			return
		}
		item.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}/get", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		itemKey := mux.Vars(r)["item_key"]
		r.ParseForm()
		format := r.Form.Get("format")

		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}
		item := dataset.GetItem(itemKey)
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

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}/get-video-frame", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		itemKey := mux.Vars(r)["item_key"]
		r.ParseForm()
		frameIdx := skyhook.ParseInt(r.Form.Get("idx"))

		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		} else if dataset.DataType != skyhook.VideoType {
			http.Error(w, "dataset is not video type", 404)
			return
		}
		item := dataset.GetItem(itemKey)
		if item == nil {
			http.Error(w, "no matching item", 404)
			return
		}

		item.Load()
		data, err := item.LoadData()
		if err != nil {
			panic(err)
		}
		reader := data.(skyhook.VideoData).ReadSlice(frameIdx, frameIdx+1)
		defer reader.Close()
		imageData, err := reader.Read(1)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		if err := imageData.Encode("jpeg", w); err != nil {
			panic(err)
		}
	})
}
