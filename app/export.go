package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"archive/zip"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
)

func getExportFilename(name string) string {
	// try name.zip, name-0.zip, name-1.zip, etc. until we find one that doesn't already exist
	outFname := filepath.Join("exports", name+".zip")
	for counter := 0; skyhook.FileExists(outFname); counter++ {
		outFname = filepath.Join("exports", fmt.Sprintf("%s-%d.zip", name, counter))
	}
	return outFname
}

// Export a dataset into the Skyhook .zip format.
// Returns the zip filename or error.
func (ds *DBDataset) Export() (string, error) {
	// get absolute path to the export filename
	// we need it to be absolute since we will run `zip` in different working directory
	outFname := getExportFilename(ds.Name)
	var err error
	outFname, err = filepath.Abs(outFname)
	if err != nil {
		return "", err
	}

	// run zip utility to produce the export
	// TODO: well sqlite3 may be in use, maybe we should export/snapshot the sqlite3 separately before zipping or something
	log.Printf("[export] beginning export of %s to %s", ds.Name, outFname)
	cmd := exec.Command("zip", "-r", outFname, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = ds.Dirname()
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return filepath.Base(outFname), nil
}

// Export files in a file dataset.
// Unlike Export, this produces a .zip file where items in the dataset are named
// based on their filenames specified in FileMetadata.
func (ds *DBDataset) ExportFiles() (string, error) {
	if ds.DataType != skyhook.FileType {
		panic(fmt.Errorf("ExportFiles called on non-file dataset"))
	}

	outFname := getExportFilename(ds.Name)

	log.Printf("[export-files] exporting File dataset %s to %s", ds.Name, outFname)
	file, err := os.Create(outFname)
	if err != nil {
		return "", err
	}
	defer file.Close()
	zipWriter := zip.NewWriter(file)
	for _, item := range ds.ListItems() {
		data, err := item.LoadData()
		if err != nil {
			return "", err
		}
		fdata := data.(skyhook.FileData)
		w, err := zipWriter.Create(fdata.Metadata.Filename)
		if err != nil {
			return "", err
		}
		if _, err := w.Write(fdata.Bytes); err != nil {
			return "", err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return "", err
	}
	return filepath.Base(outFname), nil
}

func init() {
	// We reuse the same handler except dataset.Export... call for both /export and /export-files
	exportHandler := func(files bool) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
			dataset := GetDataset(dsID)
			if dataset == nil {
				http.Error(w, "no such dataset", 404)
				return
			}

			var fname string
			var err error
			if files {
				log.Printf("[export-files] user requested export of dataset %s", dataset.Name)
				fname, err = dataset.ExportFiles()
			} else {
				log.Printf("[export] user requested export of dataset %s", dataset.Name)
				fname, err = dataset.Export()
			}

			if err != nil {
				log.Printf("[export] export failed: %v", err)
				http.Error(w, err.Error(), 400)
				return
			}
			skyhook.JsonResponse(w, fname)
		}
	}

	Router.HandleFunc("/datasets/{ds_id}/export", exportHandler(false)).Methods("POST")
	Router.HandleFunc("/datasets/{ds_id}/export-files", exportHandler(true)).Methods("POST")

	fileServer := http.FileServer(http.Dir("exports/"))
	Router.PathPrefix("/exports/").Handler(http.StripPrefix("/exports/", fileServer))
}
