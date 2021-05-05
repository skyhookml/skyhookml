package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (item *DBItem) Handle(format string, w http.ResponseWriter, r *http.Request) {
	item.Load()

	fname := item.Fname()
	if format == item.Format && fname != "" {
		http.ServeFile(w, r, fname)
		return
	}

	data, metadata, err := item.LoadData()
	if err != nil {
		panic(err)
	}

	if format == "jpeg" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else if format == "png" {
		w.Header().Set("Content-Type", "image/png")
	} else if format == "mp4" {
		w.Header().Set("Content-Type", "video/mp4")
	} else if format == "json" {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		var filename string
		if item.Dataset.DataType == skyhook.FileType {
			filename = metadata.(skyhook.FileMetadata).Filename
		} else {
			ext := skyhook.GetExtFromFormat(item.Dataset.DataType, format)
			if ext == "" {
				ext = item.Ext
			}
			filename = item.Key + "." + ext
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=\"%s\"", filename))
	}

	if err := item.DataSpec().Write(data, format, metadata, w); err != nil {
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

		var request DatasetUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		dataset.Update(request)
	}).Methods("POST")

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

		item_, err := dataset.AddItem(item)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		skyhook.JsonResponse(w, item_)
	}).Methods("POST")

	// handle endpoints starting with /datasets/{ds_id}/items/{item_key}
	handleItem := func(f func(http.ResponseWriter, *http.Request, *DBDataset, *DBItem)) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
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
			f(w, r, dataset, item)
		}
	}

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}", handleItem(func(w http.ResponseWriter, r *http.Request, dataset *DBDataset, item *DBItem) {
		skyhook.JsonResponse(w, item)
	})).Methods("GET")

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}", handleItem(func(w http.ResponseWriter, r *http.Request, dataset *DBDataset, item *DBItem) {
		item.Delete()
	})).Methods("DELETE")

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}/get", handleItem(func(w http.ResponseWriter, r *http.Request, dataset *DBDataset, item *DBItem) {
		r.ParseForm()
		format := r.Form.Get("format")

		if format == "meta" {
			metadata := item.DecodeMetadata()
			skyhook.JsonResponse(w, metadata)
			return
		}

		item.Handle(format, w, r)
	}))

	Router.HandleFunc("/datasets/{ds_id}/items/{item_key}/get-video-frame", handleItem(func(w http.ResponseWriter, r *http.Request, dataset *DBDataset, item *DBItem) {
		r.ParseForm()
		frameIdx := skyhook.ParseInt(r.Form.Get("idx"))

		if dataset.DataType != skyhook.VideoType {
			http.Error(w, "dataset is not video type", 404)
			return
		}

		item.Load()
		videoSpec := skyhook.DataSpecs[skyhook.VideoType].(skyhook.VideoDataSpec)
		imageSpec := skyhook.DataSpecs[skyhook.ImageType]
		reader := videoSpec.ReadSlice("mp4", item.DecodeMetadata(), item.Fname(), frameIdx, frameIdx+1)
		defer reader.Close()
		data, err := reader.Read(1)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading frame: %v", err), 400)
			return
		}
		images := data.([]skyhook.Image)
		w.Header().Set("Content-Type", "image/jpeg")
		if err := imageSpec.Write(images[0], "jpeg", nil, w); err != nil {
			panic(err)
		}
	}))
}
