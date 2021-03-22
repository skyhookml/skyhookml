package app

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Import a local path that contains a Skyhook dataset.
// If symlink is true, we will copy the db.sqlite3 but symlink all the other
//   files in the dataset. We do not symlink the whole directory currently
//   because we don't want to modify it.
// If symlink is false, we just copy all the files.
func ImportDataset(path string, symlink bool) error {
	// first figure out the dataset attributes from the sqlite3 file
	srcDBFname := filepath.Join(path, "db.sqlite3")
	dsdb := GetCachedDB(srcDBFname, nil)
	if dsdb == nil {
		return fmt.Errorf("error opening db.sqlite3 in %s", path)
	}
	var rawds skyhook.Dataset
	dsdb.QueryRow("SELECT name, type, data_type, hash FROM datasets").Scan(&rawds.Name, &rawds.Type, &rawds.DataType, &rawds.Hash)
	UncacheDB(srcDBFname)

	// create a new dataset and the directory, and copy the sqlite3
	ds := NewDataset(rawds.Name, rawds.Type, rawds.DataType, rawds.Hash)
	ds.Mkdir()
	if err := skyhook.CopyFile(srcDBFname, ds.DBFname()); err != nil {
		ds.Delete()
		return fmt.Errorf("error copying sqlite3 %s: %v", srcDBFname, err)
	}

	// now copy or symlink all the other files
	for _, item := range ds.ListItems() {
		dstFname := item.Fname()
		srcFname := filepath.Join(path, filepath.Base(dstFname))
		err := skyhook.CopyOrSymlink(srcFname, dstFname, symlink)
		if err != nil {
			ds.Delete()
			return fmt.Errorf("error adding %s: %v", srcFname, err)
		}
	}

	return nil
}

func GetKeyFromFilename(fname string) string {
	for i := len(fname)-1; i >= 0; i-- {
		if fname[i] == '.' {
			return fname[0:i]
		}
	}
	// no dot, return whole filename
	return fname
}

// specialized function for importing files if the dataset is File type
// in this case, the metadata specifies the original filename
// also in this case we want to recursively scan since filenames in subdirectories should be imported too
func (ds *DBDataset) ImportIntoFileDataset(fnames []string, symlink bool) error {
	// set of keys that have been used already
	keySet := make(map[string]bool)
	// get an unused key from a relative path
	// and update keySet
	getKey := func(path string) string {
		key := strings.ReplaceAll(path, string(os.PathSeparator), "_")
		// we should've replaced path separators above, but take base just in case it didn't work right
		key = filepath.Base(key)
		// remove the extension, if any
		ext := filepath.Ext(key)
		key = key[0:len(key)-len(ext)]
		// make sure the key isn't empty now
		if key == "" {
			key = "x"
		}
		// resolve conflicts
		startKey := key
		for counter := 0; keySet[key]; counter++ {
			key = fmt.Sprintf("%s-%d", startKey, counter)
		}

		keySet[key] = true
		return key
	}

	// walk over all files
	for _, root := range fnames {
		// determine base directory for computing relative paths
		var rootDir string
		fi, err := os.Stat(root)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			rootDir = root
		} else {
			rootDir = filepath.Dir(root)
		}

		err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				// currently we always stop if there was some error walking
				// it could potentially be due to special directory though
				return err
			}

			// don't need to do anything for directories
			if info.IsDir() {
				return nil
			}

			curPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return fmt.Errorf("error computing relative path: %v", err)
			}
			key := getKey(curPath)

			baseName := info.Name()
			ext := filepath.Ext(baseName)
			if len(ext) > 0 && ext[0] == '.' {
				ext = ext[1:]
			} else if ext == "" {
				ext = "file"
			}

			metadata := skyhook.FileMetadata{
				Filename: curPath,
			}
			item := ds.AddItem(skyhook.Item{
				Key: key,
				Ext: ext,
				Format: "",
				Metadata: string(skyhook.JsonMarshal(metadata)),
			})
			item.Mkdir()

			err = skyhook.CopyOrSymlink(path, item.Fname(), symlink)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DBDataset) ImportFiles(fnames []string) error {
	if ds.DataType == skyhook.FileType {
		return ds.ImportIntoFileDataset(fnames, false)
	}

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
		if err := skyhook.CopyFile(fname, item.Fname()); err != nil {
			return err
		}

		if err := item.SetMetadataFromFile(); err != nil {
			return err
		}
	}

	return nil
}

func (ds *DBDataset) ImportDir(path string) error {
	if ds.DataType == skyhook.FileType {
		return ds.ImportIntoFileDataset([]string{path}, false)
	}

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
func HandleUpload(w http.ResponseWriter, r *http.Request, f func(fname string, cleanupFunc func()) error) {
	err := func() error {
		file, fh, err := r.FormFile("file")
		if err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		// write file to a temporary file on disk with same filename
		tmpdir, err := ioutil.TempDir("", "upload")
		if err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		basename := filepath.Base(fh.Filename)
		tmpfname := filepath.Join(tmpdir, basename)
		tmpfile, err := os.Create(tmpfname)
		if err != nil {
			return fmt.Errorf("error processing upload: %v", err)
		}
		cleanup := func() {
			os.RemoveAll(tmpdir)
		}
		if _, err := io.Copy(tmpfile, file); err != nil {
			cleanup()
			return fmt.Errorf("error processing upload: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			cleanup()
			return fmt.Errorf("error processing upload: %v", err)
		}
		return f(tmpfile.Name(), cleanup)
	}()
	if err != nil {
		log.Printf("[upload %s] error: %v", r.URL.Path, err)
		http.Error(w, err.Error(), 400)
	}
}

func init() {
	Router.HandleFunc("/import-dataset", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		mode := r.PostForm.Get("mode")
		forceCopy := r.PostForm.Get("forcecopy") == "true"

		importFunc := func(path string) {
			err := func() error {
				if strings.HasSuffix(path, ".zip") {
					log.Printf("[import-dataset] importing zip file [%s]", path)
					return UnzipThen(path, func(path string) error  {
						return ImportDataset(path, false)
					})
				}
				if fi, statErr := os.Stat(path); statErr == nil && fi.IsDir() {
					log.Printf("[import-dataset] importing directory [%s]", path)
					return ImportDataset(path, !forceCopy)
				}
				return fmt.Errorf("import-dataset expected zip file or directory, but [%s] is neither", path)
			}()
			if err == nil {
				log.Printf("[import-dataset] ... import from %s succeeded", path)
			} else {
				log.Printf("[import-dataset] ... import from %s failed: %v", path, err)
			}
		}

		if mode == "local" {
			path := r.PostForm.Get("path")
			go importFunc(path)
		} else if mode == "upload" {
			log.Printf("[import-dataset] handling import from upload request")
			HandleUpload(w, r, func(fname string, cleanup func()) error {
				log.Printf("[import-dataset] importing from upload request: %s", fname)
				go func() {
					importFunc(fname)
					cleanup()
				}()
				return nil
			})
		}
	}).Methods("POST")

	Router.HandleFunc("/datasets/{ds_id}/import-upload", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}

		log.Printf("[import-upload] handling import from upload request")
		HandleUpload(w, r, func(fname string, cleanup func()) error {
			log.Printf("[import-upload] importing from upload request: %s", fname)

			go func() {
				defer cleanup()
				var err error
				if strings.HasSuffix(fname, ".zip") {
					err = UnzipThen(fname, dataset.ImportDir)
				} else {
					err = dataset.ImportFiles([]string{fname})
				}
				if err != nil {
					log.Printf("[import-upload] failed on %s: %v", fname, err)
				}
			}()
			return nil
		})
	})

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
		HandleUpload(w, r, func(fname string, cleanup func()) error {
			log.Printf("[import-upload] importing from upload request: %s", fname)
			go func() {
				defer cleanup()
				var err error
				if strings.HasSuffix(fname, ".zip") {
					err = UnzipThen(fname, dataset.ImportDir)
				} else {
					err = dataset.ImportFiles([]string{fname})
				}
				if err != nil {
					log.Printf("[import-upload] failed on %s: %v", fname, err)
				}
			}()
			return nil
		})
	})
}
