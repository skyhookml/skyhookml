package convert

import (
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"path/filepath"
	"strings"
)

// Convert to and from YOLOv3 format.
// Skyhook inputs requires two datasets, one image and one detection.
// This format is a flat FileDataset with paired images and labels stored under same original filename.
// An obj.names file is also created for the category names.

func init() {
	imageImpl := skyhook.DataImpls[skyhook.ImageType]

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "to_yolo",
			Name: "To YOLO",
			Description: "Convert from [image, detection] datasets to YOLO image/txt format",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "detections", DataTypes: []skyhook.DataType{skyhook.DetectionType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "output", DataType: skyhook.FileType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// we mostly use SimpleTasks, which creates a task for each corresponding image/detection pair between the input datasets
			// but we need to assign one task for writing the "obj.names" output
			// to assign it, we just set the metadata to "obj.names", which applyFunc below will check
			tasks, err := exec_ops.SimpleTasks(node, rawItems)
			if err != nil {
				return nil, err
			}
			if len(tasks) > 0 {
				tasks[0].Metadata =  "obj.names"
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params struct {
				Format string
				Symlink bool
			}
			if err := exec_ops.DecodeParams(node, &params, true); err != nil {
				return nil, err
			}
			if node.Params == "" {
				params.Format = "jpeg"
			}

			outDS := node.OutputDatasets["output"]
			applyFunc := func(task skyhook.ExecTask) error {
				inImageItem := task.Items["images"][0][0]
				inLabelItem := task.Items["detections"][0][0]

				// write the image
				// we produce a symlink if requested by the user and if the output format matches
				// if the output format doesn't match, we have to decode and re-encode the image
				outImageFormat := params.Format
				if outImageFormat == "" {
					outImageFormat = inImageItem.Format
				}
				outImageExt := imageImpl.GetExtGivenFormat(outImageFormat)
				if outImageExt == "" {
					// unknown format...? just use the format as ext
					outImageExt = outImageFormat
				}
				outImageMetadata := string(skyhook.JsonMarshal(skyhook.FileMetadata{
					Filename: task.Key+"."+outImageExt,
				}))
				outImageItem, err := exec_ops.AddItem(url, outDS, task.Key+"-image", outImageExt, "", outImageMetadata)
				if err != nil {
					return err
				}
				err = inImageItem.CopyTo(outImageItem.Fname(), outImageFormat, params.Symlink)
				if err != nil {
					return err
				}

				// write the labels
				// we need to convert coordinates and also change category string to category ID
				data, err := inLabelItem.LoadData()
				if err != nil {
					return err
				}
				labelData := data.(skyhook.DetectionData)
				canvasDims := labelData.Metadata.CanvasDims
				categoryToID := make(map[string]int)
				for i, category := range labelData.Metadata.Categories {
					categoryToID[category] = i
				}
				var lines []string
				for _, detection := range labelData.Detections[0] {
					cx := float64(detection.Left+detection.Right)/2/float64(canvasDims[0])
					cy := float64(detection.Top+detection.Bottom)/2/float64(canvasDims[1])
					width := float64(detection.Right-detection.Left)/float64(canvasDims[0])
					height := float64(detection.Bottom-detection.Top)/float64(canvasDims[1])
					catID := categoryToID[detection.Category] // default to 0 if not found
					line := fmt.Sprintf("%v %v %v %v %v", catID, cx, cy, width, height)
					lines = append(lines, line)
				}
				bytes := []byte(strings.Join(lines, "\n")+"\n")
				outFileData := skyhook.FileData{
					Bytes: bytes,
					Metadata: skyhook.FileMetadata{
						Filename: task.Key+".txt",
					},
				}
				err = exec_ops.WriteItem(url, outDS, task.Key+"-label", outFileData)
				if err != nil {
					return err
				}

				// we may also need to write obj.names, if this is the one task assigned to it
				if task.Metadata == "obj.names" {
					bytes := []byte(strings.Join(labelData.Metadata.Categories, "\n")+"\n")
					fileData := skyhook.FileData{
						Bytes: bytes,
						Metadata: skyhook.FileMetadata{
							Filename: "obj.names",
						},
					}
					err := exec_ops.WriteItem(url, outDS, "obj.names", fileData)
					if err != nil {
						return err
					}
				}

				return nil
			}

			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "from_yolo",
			Name: "From YOLO",
			Description: "Convert from YOLO image/txt format to [image, detection] datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.FileType}}},
		Outputs: []skyhook.ExecOutput{
			{Name: "images", DataType: skyhook.ImageType},
			{Name: "detections", DataType: skyhook.DetectionType},
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			files := ItemsToFileMap(rawItems["input"][0], true)

			// first load obj.names to get object categories
			// we will pass it to tasks in task metadata
			var categories []string
			for _, fname := range []string{"obj.names", "label_map.txt", "labels.txt"} {
				if item, ok := files[fname]; ok {
					 data, err := item.LoadData()
					 if err != nil {
						 return nil, fmt.Errorf("from_yolo: error loading categories: %v", err)
					 }
					 fileData := data.(skyhook.FileData)
					 for _, line := range strings.Split(string(fileData.Bytes), "\n") {
						 line = strings.TrimSpace(line)
						 if line == "" {
							 continue
						 }
						 categories = append(categories, line)
					 }
					 break
				}
			}
			taskMetadata := string(skyhook.JsonMarshal(categories))

			// now create one task for each .jpg/.jpeg/.png file that has corresponding .txt
			var tasks []skyhook.ExecTask
			for fname, item := range files {
				ext := filepath.Ext(fname)
				if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
					continue
				}
				prefix := fname[0:len(fname)-len(ext)]
				labelFname := prefix+".txt"
				if _, ok := files[labelFname]; !ok {
					return nil, fmt.Errorf("from_yolo: could not find labels for image %s", fname)
				}
				tasks = append(tasks, skyhook.ExecTask{
					Key: prefix,
					Items: map[string][][]skyhook.Item{
						"image": {{item}},
						"detections": {{files[labelFname]}},
					},
					Metadata: taskMetadata,
				})
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
			labelDS := node.OutputDatasets["detections"]
			applyFunc := func(task skyhook.ExecTask) error {
				inImageItem := task.Items["image"][0][0]
				inLabelItem := task.Items["detections"][0][0]
				var categories []string
				skyhook.JsonUnmarshal([]byte(task.Metadata), &categories)

				// read first few bytes of image to get the dimensions
				// default to 720p, anyway we store it in canvas dims
				dims := [2]int{1280, 720}
				if inImageItem.Fname() != "" {
					imDims, err := skyhook.GetImageDimsFromFile(inImageItem.Fname())
					if err == nil {
						dims = imDims
					}
				}

				// convert the labels .txt file to skyhook detection format
				inLabelData, err := inLabelItem.LoadData()
				if err != nil {
					return err
				}
				var detections []skyhook.Detection
				for _, line := range strings.Split(string(inLabelData.(skyhook.FileData).Bytes), "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					parts := strings.Split(line, " ")
					clsID := skyhook.ParseInt(parts[0])
					cx := skyhook.ParseFloat(parts[1])
					cy := skyhook.ParseFloat(parts[2])
					width := skyhook.ParseFloat(parts[3])
					height := skyhook.ParseFloat(parts[4])

					var category string
					if clsID >= 0 && clsID < len(categories) {
						category = categories[clsID]
					}

					detections = append(detections, skyhook.Detection{
						Category: category,
						Left: int((cx-width/2)*float64(dims[0])),
						Top: int((cy-height/2)*float64(dims[1])),
						Right: int((cx+width/2)*float64(dims[0])),
						Bottom: int((cy+height/2)*float64(dims[1])),
					})
				}

				// add the detections
				outLabelData := skyhook.DetectionData{
					Detections: [][]skyhook.Detection{detections},
					Metadata: skyhook.DetectionMetadata{
						CanvasDims: dims,
						Categories: categories,
					},
				}
				err = exec_ops.WriteItem(url, labelDS, task.Key, outLabelData)
				if err != nil {
					return err
				}

				// add the image
				// we use the original filename to determine skyhook ext/format
				var imageFileMetadata skyhook.FileMetadata
				skyhook.JsonUnmarshal([]byte(inImageItem.Metadata), &imageFileMetadata)
				format, _, _ := imageImpl.GetDefaultMetadata(imageFileMetadata.Filename)
				ext := imageImpl.GetExtGivenFormat(format)
				outImageItem, err := exec_ops.AddItem(url, imageDS, task.Key, ext, format, "")
				if err != nil {
					return err
				}
				err = inImageItem.CopyTo(outImageItem.Fname(), format, params.Symlink)
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
