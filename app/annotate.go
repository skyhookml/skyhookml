package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"bytes"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func NewAnnotateDataset(dataset skyhook.Dataset, inputs []skyhook.ExecParent, tool string, params string) (*DBAnnotateDataset, error) {
	res := db.Exec(
		"INSERT INTO annotate_datasets (dataset_id, inputs, tool, params) VALUES (?, ?, ?, ?)",
		dataset.ID, string(skyhook.JsonMarshal(inputs)), tool, params,
	)
	return GetAnnotateDataset(res.LastInsertId()), nil
}

// info needed to annotate one item, which may or may not be present in the destination dataset
type AnnotateResponse struct {
	// The key that we're labeling.
	// May be an existing key in the destination dataset, or a new key.
	Key string

	IsExisting bool
}

func init() {
	Router.HandleFunc("/annotate-datasets", func(w http.ResponseWriter, r *http.Request) {
		l := []skyhook.AnnotateDataset{}
		for _, ds := range ListAnnotateDatasets() {
			ds.Load()
			l = append(l, ds.AnnotateDataset)
		}
		skyhook.JsonResponse(w, l)
	}).Methods("GET")

	Router.HandleFunc("/annotate-datasets", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			DatasetID int
			Inputs []skyhook.ExecParent
			Tool string
			Params string
		}
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		dataset := GetDataset(request.DatasetID)
		ds, err := NewAnnotateDataset(dataset.Dataset, request.Inputs, request.Tool, request.Params)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		skyhook.JsonResponse(w, ds)
	}).Methods("POST")

	Router.HandleFunc("/annotate-datasets/{s_id}", func(w http.ResponseWriter, r *http.Request) {
		sID := skyhook.ParseInt(mux.Vars(r)["s_id"])
		annoset := GetAnnotateDataset(sID)
		if annoset == nil {
			http.Error(w, "no such annotate dataset", 404)
			return
		}
		annoset.Load()
		skyhook.JsonResponse(w, annoset)
	}).Methods("GET")

	Router.HandleFunc("/annotate-datasets/{s_id}", func(w http.ResponseWriter, r *http.Request) {
		sID := skyhook.ParseInt(mux.Vars(r)["s_id"])
		annoset := GetAnnotateDataset(sID)
		if annoset == nil {
			http.Error(w, "no such annotate dataset", 404)
			return
		}

		var request AnnotateDatasetUpdate
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		annoset.Update(request)
	}).Methods("POST")

	Router.HandleFunc("/annotate-datasets/{s_id}", func(w http.ResponseWriter, r *http.Request) {
		sID := skyhook.ParseInt(mux.Vars(r)["s_id"])
		annoset := GetAnnotateDataset(sID)
		if annoset == nil {
			http.Error(w, "no such annotate dataset", 404)
			return
		}
		annoset.Delete()
	}).Methods("DELETE")

	Router.HandleFunc("/annotate-datasets/{s_id}/annotate", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		sID := skyhook.ParseInt(mux.Vars(r)["s_id"])
		key := r.Form.Get("key")
		annoset := GetAnnotateDataset(sID)
		if annoset == nil {
			http.Error(w, "no such annotate dataset", 404)
			return
		}
		annoset.Load()

		// if key is not set, sample a key common across inputs that hasn't been annotated yet
		// then, set input item IDs and other params in response struct
		var resp AnnotateResponse

		if key == "" {
			key = annoset.SampleMissingKey()
			if key == "" {
				http.Error(w, "everything has been labeled already", 400)
				return
			}
			resp.Key = key
			resp.IsExisting = false
		} else {
			item := (&DBDataset{Dataset: annoset.Dataset}).GetItem(key)
			if item == nil {
				http.Error(w, "no item with key in annotate dataset", 404)
				return
			}
			resp.Key = key
			resp.IsExisting = true
		}

		skyhook.JsonResponse(w, resp)
	}).Methods("GET")

	Router.HandleFunc("/annotate-datasets/{s_id}/annotate", func(w http.ResponseWriter, r *http.Request) {
		sID := skyhook.ParseInt(mux.Vars(r)["s_id"])
		annoset := GetAnnotateDataset(sID)
		if annoset == nil {
			http.Error(w, "no such annotate dataset", 404)
			return
		}
		annoset.Load()

		type AnnotateRequest struct {
			Key string
			Data string
			Format string
			Metadata string
		}
		var request AnnotateRequest
		if err := skyhook.ParseJsonRequest(w, r, &request); err != nil {
			return
		}

		ds := &DBDataset{Dataset: annoset.Dataset}
		item := ds.GetItem(request.Key)
		metadata := ds.DataSpec().DecodeMetadata(request.Metadata)
		buf := bytes.NewBuffer([]byte(request.Data))
		data, err := ds.DataSpec().Read(request.Format, metadata, buf)
		if err != nil {
			http.Error(w, fmt.Sprintf("error decoding data: %v", err), 400)
			return
		}

		if item == nil {
			// new key
			_, err := ds.WriteItem(request.Key, data, metadata)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		} else {
			item.SetMetadata(request.Format, metadata)
			err := item.UpdateData(data, metadata)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}
	}).Methods("POST")
}
