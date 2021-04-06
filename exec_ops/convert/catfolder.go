package convert

import (
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"path/filepath"
	"sort"
	"strconv"
)

// Convert to and from image classification folders format.
// Here we just have one folder per category, and put images into folder based on their category.

func init() {
	imageImpl := skyhook.DataImpls[skyhook.ImageType]

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "to_catfolder",
			Name: "To Category-Folders",
			Description: "Convert from [image, int] datasets to Category-Folders format",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "labels", DataTypes: []skyhook.DataType{skyhook.IntType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "output", DataType: skyhook.FileType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params struct {
				Symlink bool
			}
			if err := exec_ops.DecodeParams(node, &params, true); err != nil {
				return nil, err
			}

			outDS := node.OutputDatasets["output"]
			applyFunc := func(task skyhook.ExecTask) error {
				inImageItem := task.Items["images"][0][0]
				inLabelItem := task.Items["labels"][0][0]

				// determine the category
				data_, err := inLabelItem.LoadData()
				if err != nil {
					return err
				}
				data := data_.(skyhook.IntData)
				x := data.Ints[0]
				var category string
				if x >= 0 && x < len(data.Metadata.Categories) {
					category = data.Metadata.Categories[x]
				} else {
					category = strconv.Itoa(x)
				}

				// write the imag
				outMetadata := string(skyhook.JsonMarshal(skyhook.FileMetadata{
					Filename: filepath.Join(category, task.Key+"."+inImageItem.Ext),
				}))
				outItem, err := exec_ops.AddItem(url, outDS, task.Key, inImageItem.Ext, "", outMetadata)
				if err != nil {
					return err
				}
				err = inImageItem.CopyTo(outItem.Fname(), inImageItem.Format, params.Symlink)
				if err != nil {
					return err
				}

				return nil
			}

			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "from_catfolder",
			Name: "From Category-Folders",
			Description: "Convert from Category-Folders format to [image, int] datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.FileType}}},
		Outputs: []skyhook.ExecOutput{
			{Name: "images", DataType: skyhook.ImageType},
			{Name: "labels", DataType: skyhook.IntType},
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// we use the file map to determine what categories are available
			// this needs to go in IntData.Metadata
			// so we then include it in the task metadata
			files := ItemsToFileMap(rawItems["input"][0], false)

			catSet := make(map[string]bool)
			for path := range files {
				lastDir := filepath.Base(filepath.Dir(path))
				if lastDir == "" || lastDir == "." {
					continue
				}
				catSet[lastDir] = true
			}
			var categories []string
			for category := range catSet {
				categories = append(categories, category)
			}
			sort.Strings(categories)
			taskMetadata := string(skyhook.JsonMarshal(categories))

			// now we can create one task per image
			tasks, err := exec_ops.SimpleTasks(node, rawItems)
			if err != nil {
				return nil, err
			}
			for i := range tasks {
				tasks[i].Metadata = taskMetadata
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params struct {
				Symlink bool
			}
			if err := exec_ops.DecodeParams(node, &params, true); err != nil {
				return nil, err
			}
			imageDS := node.OutputDatasets["images"]
			labelDS := node.OutputDatasets["labels"]
			applyFunc := func(task skyhook.ExecTask) error {
				inItem := task.Items["input"][0][0]
				var categories []string
				skyhook.JsonUnmarshal([]byte(task.Metadata), &categories)

				catSet := make(map[string]int)
				for i, category := range categories {
					catSet[category] = i
				}

				// extract category from folder
				// also extract the filename without extension
				var metadata skyhook.FileMetadata
				skyhook.JsonUnmarshal([]byte(inItem.Metadata), &metadata)
				category := filepath.Base(filepath.Dir(metadata.Filename))
				x, ok := catSet[category]
				if !ok {
					return fmt.Errorf("unknown category %s from filename %s", category, metadata.Filename)
				}

				fname := filepath.Base(metadata.Filename)
				key := fname[0:len(fname)-len(filepath.Ext(fname))]

				// copy the image
				format, _, _ := imageImpl.GetDefaultMetadata(fname)
				ext := imageImpl.GetExtGivenFormat(format)
				outImageItem, err := exec_ops.AddItem(url, imageDS, key, ext, format, "")
				if err != nil {
					return err
				}
				err = inItem.CopyTo(outImageItem.Fname(), format, params.Symlink)
				if err != nil {
					return err
				}

				// add the labels
				outLabelData := skyhook.IntData{
					Ints: []int{x},
					Metadata: skyhook.IntMetadata{
						Categories: categories,
					},
				}
				err = exec_ops.WriteItem(url, labelDS, key, outLabelData)
				if err != nil {
					return err
				}

				return nil
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
