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
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type ImportOptions struct {
	Symlink bool

	// import will call Update and IsStopping on this AppJobOp if set
	AppJobOp *AppJobOp

	// import will call Increment if set
	ProgressJobOp *ProgressJobOp
}

func (opts ImportOptions) SetTasks(total int) {
	if opts.ProgressJobOp != nil {
		opts.ProgressJobOp.SetTotal(total)
	}
}

// increment the ProgressJobOp, write a line to AppJobOp, and check IsStopping
func (opts ImportOptions) CompletedTask(line string, increment int) bool {
	if increment > 0 && opts.ProgressJobOp != nil {
		for i := 0; i < increment; i++ {
			opts.ProgressJobOp.Increment()
		}
	}
	if opts.AppJobOp != nil {
		if line != "" {
			opts.AppJobOp.Update([]string{line})
		} else if increment > 0 {
			// Empty update to make sure progress increment gets reflected.
			opts.AppJobOp.Update(nil)
		}
		if opts.AppJobOp.IsStopping() {
			return true
		}
	}
	return false
}

// Import a local path that contains a Skyhook dataset.
// If symlink is true, we will copy the db.sqlite3 but symlink all the other
//   files in the dataset. We do not symlink the whole directory currently
//   because we don't want to modify it.
// If symlink is false, we just copy all the files.
func ImportDataset(path string, opts ImportOptions) error {
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
	opts.AppJobOp.Job.UpdateMetadata(strconv.Itoa(ds.ID))
	ds.Mkdir()
	if err := skyhook.CopyFile(srcDBFname, ds.DBFname()); err != nil {
		ds.Delete()
		return fmt.Errorf("error copying sqlite3 %s: %v", srcDBFname, err)
	}

	// now copy or symlink all the other files
	items := ds.ListItems()
	opts.SetTasks(len(items))
	for _, item := range items {
		dstFname := item.Fname()
		srcFname := filepath.Join(path, filepath.Base(dstFname))
		err := skyhook.CopyOrSymlink(srcFname, dstFname, opts.Symlink)
		if err != nil {
			ds.Delete()
			return fmt.Errorf("error adding %s: %v", srcFname, err)
		}

		stopping := opts.CompletedTask(fmt.Sprintf("Copied %s to %s", srcFname, dstFname), 1)
		if stopping {
			return fmt.Errorf("stopped by user")
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
func (ds *DBDataset) ImportIntoFileDataset(fnames []string, opts ImportOptions) error {
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
	ds.Mkdir()
	opts.SetTasks(len(fnames))
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

			// we use the path of this file relative to the base directory as the FileMetadata.Filename
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
			item, err := ds.AddItem(skyhook.Item{
				Key: key,
				Ext: ext,
				Format: "",
				Metadata: string(skyhook.JsonMarshal(metadata)),
			})
			if err != nil {
				return err
			}

			err = skyhook.CopyOrSymlink(path, item.Fname(), opts.Symlink)
			if err != nil {
				return err
			}

			// log to job console if any
			// note that we're not updating progress here since we don't know the number of files a priori
			stopping := opts.CompletedTask(fmt.Sprintf("Imported [%s]", curPath), 0)
			if stopping {
				return fmt.Errorf("stopped by user")
			}

			return nil
		})

		if err != nil {
			return err
		}

		opts.CompletedTask("", 1)
	}

	return nil
}

func (ds *DBDataset) ImportFiles(fnames []string, opts ImportOptions) error {
	if ds.DataType == skyhook.FileType {
		return ds.ImportIntoFileDataset(fnames, opts)
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

	ds.Mkdir()
	opts.SetTasks(len(fnames))
	for _, fname := range fnames {
		key := GetKeyFromFilename(filepath.Base(fname))
		ext := filepath.Ext(fname)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		item, err := ds.AddItem(skyhook.Item{
			Key: key,
			Ext: ext,
			Format: "",
			Metadata: "",
		})
		if err != nil {
			return err
		}

		// copy the file
		if err := skyhook.CopyOrSymlink(fname, item.Fname(), opts.Symlink); err != nil {
			return err
		}

		if err := item.SetMetadataFromFile(); err != nil {
			return err
		}

		stopping := opts.CompletedTask(fmt.Sprintf("Imported %s", fname), 1)
		if stopping {
			return fmt.Errorf("stopped by user")
		}
	}

	return nil
}

func (ds *DBDataset) ImportDir(path string, opts ImportOptions) error {
	if ds.DataType == skyhook.FileType {
		return ds.ImportIntoFileDataset([]string{path}, opts)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	var fnames []string
	for _, fi := range files {
		fnames = append(fnames, filepath.Join(path, fi.Name()))
	}
	return ds.ImportFiles(fnames, opts)
}

// Import from a URL.
// Calls handler function after URL is downloaded and unzipped.
// Updates opts with progress.
func ImportURL(url string, opts ImportOptions, f func(path string) error) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("error getting %s: status %d", url, resp.StatusCode)
	}
	file, err := ioutil.TempFile("", "*.zip")
	if err != nil {
		return fmt.Errorf("error making temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	// download the file and update opts with progress
	opts.AppJobOp.Update([]string{fmt.Sprintf("Downloading from %s to %s", url, file.Name())})
	buf := make([]byte, 4096)
	read := 0
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			file.Write(buf[0:n])
		}
		read += n
		if resp.ContentLength > 0 {
			updated := opts.ProgressJobOp.SetProgressPercent(read * 100 / int(resp.ContentLength))
			if updated {
				opts.AppJobOp.Update(nil)
			}
		}
		if err == nil {
			continue
		} else if err == io.EOF {
			break
		}
	}
	opts.AppJobOp.Update([]string{"Download completed!"})

	return UnzipThen(file.Name(), f)
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
		"unzip", "-d", tmpDir, fname,
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
	// Helper function to create ImportOptions.
	// Mostly we need to create a Job for the import operation.
	makeImportOptions := func(name string, symlink bool) ImportOptions {
		job := NewJob(
			name,
			"import",
			"consoleprogress",
			"",
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
		return ImportOptions{
			Symlink: symlink,
			AppJobOp: jobOp,
			ProgressJobOp: progressJobOp,
		}
	}

	// Importing a dataset in SkyhookML archive format.
	Router.HandleFunc("/import-dataset", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		mode := r.Form.Get("mode")
		symlink := r.Form.Get("symlink") == "true"

		importFunc := func(path string, opts ImportOptions) {
			err := func() error {
				if strings.HasSuffix(path, ".zip") {
					log.Printf("[import-dataset] importing zip file [%s]", path)
					return UnzipThen(path, func(path string) error  {
						opts.Symlink = false
						return ImportDataset(path, opts)
					})
				}
				if fi, statErr := os.Stat(path); statErr == nil && fi.IsDir() {
					log.Printf("[import-dataset] importing directory [%s]", path)
					return ImportDataset(path, opts)
				}
				return fmt.Errorf("import-dataset expected zip file or directory, but [%s] is neither", path)
			}()
			if err == nil {
				log.Printf("[import-dataset] ... import from %s succeeded", path)
			} else {
				log.Printf("[import-dataset] ... import from %s failed: %v", path, err)
			}
			opts.AppJobOp.SetDone(err)
		}

		if mode == "local" {
			path := r.PostForm.Get("path")
			opts := makeImportOptions(fmt.Sprintf("Import from %s", path), symlink)
			go importFunc(path, opts)
			skyhook.JsonResponse(w, opts.AppJobOp.Job)
		} else if mode == "upload" {
			log.Printf("[import-dataset] handling import from upload request")
			HandleUpload(w, r, func(fname string, cleanup func()) error {
				log.Printf("[import-dataset] importing from upload request: %s", fname)
				opts := makeImportOptions(fmt.Sprintf("Import Uploaded Dataset [%s]", fname), false)
				go func() {
					importFunc(fname, opts)
					cleanup()
				}()
				skyhook.JsonResponse(w, opts.AppJobOp.Job)
				return nil
			})
		} else if mode == "url" {
			url := r.PostForm.Get("url")
			opts := makeImportOptions(fmt.Sprintf("Import from %s", url), false)
			go func() {
				err := ImportURL(url, opts, func(path string) error {
					importFunc(path, opts)
					return nil
				})
				if err != nil {
					// This means we didn't quite make it to importFunc.
					// So we need to set the error here.
					log.Printf("[import-dataset] failed to download %s: %v", url)
					opts.AppJobOp.SetDone(err)
				}
			}()
			skyhook.JsonResponse(w, opts.AppJobOp.Job)
		}
	}).Methods("POST")

	// Adding new files to an existing dataset.
	Router.HandleFunc("/datasets/{ds_id}/import", func(w http.ResponseWriter, r *http.Request) {
		dsID := skyhook.ParseInt(mux.Vars(r)["ds_id"])
		dataset := GetDataset(dsID)
		if dataset == nil {
			http.Error(w, "no such dataset", 404)
			return
		}

		r.ParseForm()
		mode := r.Form.Get("mode")
		symlink := r.Form.Get("symlink") == "true"

		importFunc := func(path string, opts ImportOptions) {
			var err error
			if strings.HasSuffix(path, ".zip") {
				log.Printf("[import] importing zip file [%s]", path)
				err = UnzipThen(path, func(path string) error {
					opts.Symlink = false
					return dataset.ImportDir(path, opts)
				})
			} else {
				if fi, statErr := os.Stat(path); statErr == nil && fi.IsDir() {
					log.Printf("[import] importing directory [%s]", path)
					err = dataset.ImportDir(path, opts)
				} else {
					log.Printf("[import] importing file [%s]", path)
					err = dataset.ImportFiles([]string{path}, opts)
				}
			}

			if err == nil {
				log.Printf("[import] ... import from %s succeeded", path)
			} else {
				log.Printf("[import] ... import from %s failed: %v", path, err)
			}
			opts.AppJobOp.SetDone(err)
		}

		jobName := fmt.Sprintf("Import Into %s", dataset.Name)
		if mode == "local" {
			path := r.PostForm.Get("path")
			opts := makeImportOptions(jobName, symlink)
			go importFunc(path, opts)
			skyhook.JsonResponse(w, opts.AppJobOp.Job)
		} else if mode == "upload" {
			log.Printf("[import] handling import from upload request")
			HandleUpload(w, r, func(fname string, cleanup func()) error {
				log.Printf("[import] importing from upload request: %s", fname)
				opts := makeImportOptions(jobName, false)
				go func() {
					importFunc(fname, opts)
					cleanup()
				}()
				skyhook.JsonResponse(w, opts.AppJobOp.Job)
				return nil
			})
		}
	}).Methods("POST")
}
