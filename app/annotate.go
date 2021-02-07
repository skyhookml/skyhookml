package app

import (
	"../skyhook"

	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func NewAnnotateDataset(dataset skyhook.Dataset, inputs []skyhook.Dataset, tool string, params string) (*DBAnnotateDataset, error) {
	inputIDs := make([]string, len(inputs))
	for i, input := range inputs {
		inputIDs[i] = strconv.Itoa(input.ID)
	}
	res := db.Exec(
		"INSERT INTO annotate_datasets (dataset_id, inputs, tool, params) VALUES (?, ?, ?, ?)",
		dataset.ID, strings.Join(inputIDs, ","), tool, params,
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
		r.ParseForm()
		dsID := skyhook.ParseInt(r.PostForm.Get("ds_id"))
		inputsStr := r.PostForm.Get("inputs")
		tool := r.PostForm.Get("tool")
		params := r.PostForm.Get("params")

		dataset := GetDataset(dsID)

		var inputs []skyhook.Dataset
		for _, inputStr := range strings.Split(inputsStr, ",") {
			inputs = append(inputs, GetDataset(skyhook.ParseInt(inputStr)).Dataset)
		}

		ds, err := NewAnnotateDataset(dataset.Dataset, inputs, tool, params)
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
		buf := bytes.NewBuffer([]byte(request.Data))
		data, err := skyhook.DecodeData(annoset.Dataset.DataType, request.Format, request.Metadata, buf)
		if err != nil {
			panic(err)
		}

		if item == nil {
			// new key
			ds.WriteItem(request.Key, data)
		} else {
			item.UpdateData(data)
		}
	}).Methods("POST")
}
