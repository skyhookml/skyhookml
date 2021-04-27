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
func (ds *DBDataset) Export(outFname string, opts ImportOptions) error {
	// get absolute path to the export filename
	// we need it to be absolute since we will run `zip` in different working directory
	var err error
	outFname, err = filepath.Abs(outFname)
	if err != nil {
		return err
	}

	// run zip utility to produce the export
	// TODO: well sqlite3 may be in use, maybe we should export/snapshot the sqlite3 separately before zipping or something
	log.Printf("[export] beginning export of %s to %s", ds.Name, outFname)
	cmd := exec.Command("zip", "-r", outFname, ".")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Dir = ds.Dirname()
	if err := cmd.Start(); err != nil {
		return err
	}
	if opts.AppJobOp != nil {
		go opts.AppJobOp.ReadFrom(stdout)
		go opts.AppJobOp.ReadFrom(stderr)
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// Export files in a file dataset.
// Unlike Export, this produces a .zip file where items in the dataset are named
// based on their filenames specified in FileMetadata.
func (ds *DBDataset) ExportFiles(outFname string, opts ImportOptions) error {
	if ds.DataType != skyhook.FileType {
		panic(fmt.Errorf("ExportFiles called on non-file dataset"))
	}

	log.Printf("[export-files] exporting File dataset %s to %s", ds.Name, outFname)

	// Initialize zip writer.
	file, err := os.Create(outFname)
	if err != nil {
		return err
	}
	defer file.Close()
	zipWriter := zip.NewWriter(file)

	// Write items one by one.
	items := ds.ListItems()
	opts.SetTasks(len(items))
	for _, item := range items {
		// We load the FileData, and write its bytes into a new file in the zip archive.
		data, err := item.LoadData()
		if err != nil {
			return err
		}
		fdata := data.(skyhook.FileData)
		w, err := zipWriter.Create(fdata.Metadata.Filename)
		if err != nil {
			return err
		}
		if _, err := w.Write(fdata.Bytes); err != nil {
			return err
		}

		// Update progress and make sure the job hasn't been terminated.
		stopping := opts.CompletedTask(fmt.Sprintf("Added %s", fdata.Metadata.Filename), 1)
		if stopping {
			return fmt.Errorf("stopped by user")
		}
	}
	if err := zipWriter.Close(); err != nil {
		return err
	}
	return nil
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

			// determine filename and create the job/ImportOptions
			outFname := getExportFilename(dataset.Name)
			job := NewJob(
				fmt.Sprintf("Export %s", dataset.Name),
				"export",
				"export",
				outFname,
			)
			progressJobOp := &ProgressJobOp{}
			jobOp := &AppJobOp{
				Job: job,
				TailOp: &skyhook.TailJobOp{},
				WrappedJobOps: map[string]skyhook.JobOp{
					"progress": progressJobOp,
				},
			}
			job.AttachOp(jobOp)
			opts := ImportOptions{
				AppJobOp: jobOp,
				ProgressJobOp: progressJobOp,
			}

			// start the export asynchronously
			log.Printf("[export] user requested export of dataset %s", dataset.Name)
			go func() {
				var err error
				if files {
					err = dataset.ExportFiles(outFname, opts)
				} else {
					err = dataset.Export(outFname, opts)
				}
				if err == nil {
					log.Printf("[export] export of %s succeeded", dataset.Name)
				} else {
					log.Printf("[export] export of %s failed: %v", dataset.Name, err)
				}
				opts.AppJobOp.SetDone(err)
			}()
			skyhook.JsonResponse(w, job)
		}
	}

	Router.HandleFunc("/datasets/{ds_id}/export", exportHandler(false)).Methods("POST")
	Router.HandleFunc("/datasets/{ds_id}/export-files", exportHandler(true)).Methods("POST")

	fileServer := http.FileServer(http.Dir("exports/"))
	Router.PathPrefix("/exports/").Handler(http.StripPrefix("/exports/", fileServer))
}
