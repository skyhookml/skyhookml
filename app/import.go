package app

import (
	"../skyhook"

	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

func GetKeyFromFilename(fname string) string {
	for i := len(fname)-1; i >= 0; i-- {
		if fname[i] == '.' {
			return fname[0:i]
		}
	}
	// no dot, return whole filename
	return fname
}

func (ds *DBDataset) ImportFiles(fnames []string) error {
	// initial pass to make sure the filenames don't conflict with existing keys
	items := ds.ListItems()
	existingKeys := make(map[string]bool)
	for _, item := range items {
		existingKeys[item.Key] = true
	}
	for _, fname := range fnames {
		key := GetKeyFromFilename(filepath.Base(fname))
		if existingKeys[key] {
			return fmt.Errorf("key %s already exists in dataset %s", key, ds.Name)
		}
	}

	for _, fname := range fnames {
		key := GetKeyFromFilename(filepath.Base(fname))
		ext := filepath.Ext(fname)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		item := ds.AddItem(skyhook.Item{
			Key: key,
			Ext: ext,
			Format: "",
			Metadata: "",
		})
		item.Mkdir()

		// copy the file
		err := func() error {
			src, err := os.Open(fname)
			if err != nil {
				return err
			}
			defer src.Close()
			dst, err := os.Create(item.Fname())
			if err != nil {
				return err
			}
			defer dst.Close()
			if _, err := io.Copy(dst, src); err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}

		if err := item.SetMetadataFromFile(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *DBDataset) ImportDir(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	var fnames []string
	for _, fi := range files {
		fnames = append(fnames, filepath.Join(path, fi.Name()))
	}
	return ds.ImportFiles(fnames)
}

// unzip the filename to a temporary directory, then call another function
// afterwards we will clear the temporary directory
func UnzipThen(fname string, f func(path string) error) error {
	tmpDir, err := ioutil.TempDir("", "unzip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	err = skyhook.Command(
		"unzip", skyhook.CommandOptions{
			NoStdin: true,
			NoStdout: true,
			OnlyDebug: true,
		},
		"unzip", "-j", "-d", tmpDir, fname,
	).Wait()
	if err != nil {
		return err
	}
	return f(tmpDir)
}

// handle parts of standard upload where we save to a temporary file with same
// extension as uploaded file
func HandleUpload(w http.ResponseWriter, r *http.Request, f func(fname string) error) {
	err := func() error {
		file, fh, err := r.FormFile("file")
		if err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		// write file to a temporary file on disk with same extension
		ext := filepath.Ext(fh.Filename)
		tmpfile, err := ioutil.TempFile("", fmt.Sprintf("*%s", ext))
		if err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		if _, err := io.Copy(tmpfile, file); err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		return f(tmpfile.Name())
	}()
	if err != nil {
		log.Printf("[upload %s] error: %v", r.URL.Path, err)
		http.Error(w, err.Error(), 400)
	}
}

func init() {
	Router.HandleFunc("/datasets/{ds_id}/import-local", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}

		r.ParseForm()
		path := r.PostForm.Get("path")

		go func() {
			var err error
			if strings.HasSuffix(path, ".zip") {
				log.Printf("[import-local] importing zip file [%s]", path)
				err = UnzipThen(path, dataset.ImportDir)
			} else {
				if fi, statErr := os.Stat(path); statErr == nil && fi.IsDir() {
					log.Printf("[import-local] importing directory [%s]", path)
					err = dataset.ImportDir(path)
				} else {
					log.Printf("[import-local] importing file [%s]", path)
					err = dataset.ImportFiles([]string{path})
				}
			}

			if err == nil {
				log.Printf("[import-local] ... import from %s succeeded", path)
			} else {
				log.Printf("[import-local] ... import from %s failed: %v", path, err)
			}
		}()
	}).Methods("POST")


	Router.HandleFunc("/datasets/{ds_id}/import-upload", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}

		log.Printf("[import-upload] handling import from upload request")
		HandleUpload(w, r, func(fname string) error {
			log.Printf("[import-upload] importing from upload request: %s", fname)

			// move the file so it won't get cleaned up by HandleUpload
			newFname := filepath.Join(os.TempDir(), fmt.Sprintf("%d%s", rand.Int63(), filepath.Ext(fname)))
			if err := os.Rename(fname, newFname); err != nil {
				return err
			}

			go func() {
				var err error
				if strings.HasSuffix(fname, ".zip") {
					err = UnzipThen(newFname, dataset.ImportDir)
				} else {
					err = dataset.ImportFiles([]string{newFname})
				}
				if err != nil {
					log.Printf("[import-upload] failed on %s: %v", fname, err)
				}
			}()
			return nil
		})
	})
}
