package convert

import (
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/skyhook"

	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// Convert to and from COCO format.
// Currently we only support object detections. And we don't support RLE yet.
// We assume annotations JSON is together with images in a flat file tree.
// It could also just be JSON in which case image dataset output will be empty.

type CocoImage struct {
	Filename string `json:"file_name"`
	Width int `json:"width"`
	Height int `json:"height"`
	ID int `json:"id"`

	// only used when encoding to COCO
	CocoURL string `json:"coco_url"` // http://images.cocodataset.org/train2017/{filename}
	License int `json:"license"` // 4
	DateCaptured string `json:"date_captured"` // 2000-01-01 00:00:00
	FlickrURL string `json:"flickr_url"` // same as CocoURL
}

// We have to use a custom struct for Segmentation because COCO is stupid and
// uses different types for the exact same field. Terrible design, COCO.
type CocoRLE struct {
	Counts []int `json:"counts"`
	Size [2]int `json:"size"`
}
type CocoSegmentation struct {
	Points [][]float64
	RLE CocoRLE
}
func (s *CocoSegmentation) MarshalJSON() ([]byte, error) {
	if s.Points != nil {
		return json.Marshal(s.Points)
	} else {
		return json.Marshal(s.RLE)
	}
}
func (s *CocoSegmentation) UnmarshalJSON(data []byte) error {
	if strings.Contains(string(data), "{") {
		return json.Unmarshal(data, &s.RLE)
	} else {
		return json.Unmarshal(data, &s.Points)
	}
}

type CocoAnnotation struct {
	ImageID int `json:"image_id"`
	Bbox [4]int `json:"bbox"`
	CategoryID int `json:"category_id"`

	// only used when encoding to COCO
	ID int `json:"id"`
	IsCrowd int `json:"iscrowd"`
	Area int `json:"area"`
	Segmentation CocoSegmentation `json:"segmentation"`
}

type CocoCategory struct {
	SuperCategory string `json:"supercategory"`
	ID int `json:"id"`
	Name string `json:"name"`
}

type CocoJSON struct {
	Images []CocoImage `json:"images"`
	Annotations []CocoAnnotation `json:"annotations"`
	Categories []CocoCategory `json:"categories"`
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "to_coco",
			Name: "To COCO",
			Description: "Convert from [image, detection] datasets to COCO image/JSON format",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "detections", DataTypes: []skyhook.DataType{skyhook.DetectionType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "output", DataType: skyhook.FileType}},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// create one task for each image
			// and one task for all detections
			var tasks []skyhook.ExecTask
			for _, itemList := range rawItems["images"] {
				for _, item := range itemList {
					tasks = append(tasks, skyhook.ExecTask{
						Key: item.Key,
						Items: map[string][][]skyhook.Item{"images": {{item}}},
					})
				}
			}
			tasks = append(tasks, skyhook.ExecTask{
				Key: "annotations",
				Items: map[string][][]skyhook.Item{"detections": rawItems["detections"]},
			})
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params struct {
				Format string
				Symlink bool
			}
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				log.Printf("warning: to_coco node is not configured, using defaults")
			}

			if params.Format == "" {
				params.Format = "jpeg"
			}

			// TODO: this should probably be a shared function in skyhook/data_image.go
			formatToExt := func(format string) string {
				if format == "jpeg" {
					return "jpg"
				} else if format == "png" {
					return "png"
				} else {
					return format
				}
			}
			outputExt := formatToExt(params.Format)

			outDS := node.OutputDatasets["output"]
			applyFunc := func(task skyhook.ExecTask) error {
				// write each image independently to filename based on key
				for _, itemList := range task.Items["images"] {
					for _, inImageItem := range itemList {
						outImageMetadata := string(skyhook.JsonMarshal(skyhook.FileMetadata{
							Filename: inImageItem.Key+"."+outputExt,
						}))
						outImageItem, err := exec_ops.AddItem(url, outDS, inImageItem.Key+"-image", outputExt, "", outImageMetadata)
						if err != nil {
							return err
						}
						err = inImageItem.CopyTo(outImageItem.Fname(), params.Format, params.Symlink)
						if err != nil {
							return err
						}
					}
				}

				// group all detections into one CSV
				if len(task.Items["detections"]) == 0 {
					return nil
				}
				var coco CocoJSON
				for _, itemList := range task.Items["detections"] {
					for _, item := range itemList {
						data_, err := item.LoadData()
						if err != nil {
							return err
						}
						data := data_.(skyhook.DetectionData)
						metadata := data.Metadata
						detections := data.Detections

						// add categories if not already populated
						if len(coco.Categories) == 0 {
							for i, category := range metadata.Categories {
								coco.Categories = append(coco.Categories, CocoCategory{
									SuperCategory: category,
									ID: i+1,
									Name: category,
								})
							}
						}
						catToID := make(map[string]int)
						for i, category := range metadata.Categories {
							catToID[category] = i+1
						}

						// add image
						imageID := len(coco.Images)+1
						image := CocoImage{
							Filename: item.Key+"."+outputExt,
							Width: metadata.CanvasDims[0],
							Height: metadata.CanvasDims[1],
							ID: imageID,
							License: 4,
							DateCaptured: "2000-01-01 00:00:00",
						}
						image.CocoURL = "http://images.cocodataset.org/train2017/"+image.Filename
						image.FlickrURL = image.CocoURL
						coco.Images = append(coco.Images, image)

						// add annotation for each detection
						for _, dlist := range detections {
							for _, detection := range dlist {
								coco.Annotations = append(coco.Annotations, CocoAnnotation{
									ImageID: imageID,
									Bbox: [4]int{detection.Left, detection.Top, detection.Right-detection.Left, detection.Bottom-detection.Top},
									CategoryID: catToID[detection.Category],
									ID: len(coco.Annotations)+1,
									IsCrowd: 0,
									Area: (detection.Right-detection.Left)*(detection.Top-detection.Bottom),
									Segmentation: CocoSegmentation{
										Points: [][]float64{{
											float64(detection.Left),
											float64(detection.Top),
											float64(detection.Right),
											float64(detection.Top),
											float64(detection.Right),
											float64(detection.Bottom),
											float64(detection.Left),
											float64(detection.Bottom),
										}},
									},
								})
							}
						}
					}
				}
				bytes := skyhook.JsonMarshal(coco)
				outFileData := skyhook.FileData{
					Bytes: bytes,
					Metadata: skyhook.FileMetadata{
						Filename: "annotations.json",
					},
				}
				err := exec_ops.WriteItem(url, outDS, "annotations", outFileData)
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
			ID: "from_coco",
			Name: "From COCO",
			Description: "Convert from COCO image/JSON format to [image, detection] datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.FileType}}},
		Outputs: []skyhook.ExecOutput{
			{Name: "images", DataType: skyhook.ImageType},
			{Name: "detections", DataType: skyhook.DetectionType},
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params struct {
				Symlink bool
			}
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				log.Printf("warning: from_coco node is not configured, using defaults")
			}
			imageDS := node.OutputDatasets["images"]
			labelDS := node.OutputDatasets["detections"]

			// convert an image filename to the key that we should store it under
			filenameToKey := func(filename string) string {
				// remove directories
				filename = filepath.Base(filename)
				// remove extension
				ext := filepath.Ext(filename)
				filename = filename[0:len(filename)-len(ext)]
				return filename
			}

			applyFunc := func(task skyhook.ExecTask) error {
				inItem := task.Items["input"][0][0]

				if inItem.Ext != "json" {
					// just copy the image
					var ext, format string
					if inItem.Ext == "jpg" || inItem.Ext == "jpeg" {
						ext = "jpg"
						format = "jpeg"
					} else if inItem.Ext == "png" {
						ext = "png"
						format = "png"
					}

					// determine key to save this file under
					var metadata skyhook.FileMetadata
					skyhook.JsonUnmarshal([]byte(inItem.Metadata), &metadata)
					key := filenameToKey(metadata.Filename)

					outImageItem, err := exec_ops.AddItem(url, imageDS, key, ext, format, "")
					if err != nil {
						return err
					}
					err = inItem.CopyTo(outImageItem.Fname(), format, params.Symlink)
					if err != nil {
						return err
					}
					return nil
				}

				// so this is JSON, means we need to populate all the detections from this item
				// (0) parse JSON
				// (1) parse categories
				// (2) group annotations by image ID
				// (3) loop over images

				data_, err := inItem.LoadData()
				if err != nil {
					return err
				}
				data := data_.(skyhook.FileData)
				var coco CocoJSON
				err = json.Unmarshal(data.Bytes, &coco)
				if err != nil {
					return fmt.Errorf("error decoding annotation JSON (%s): %v", data.Metadata.Filename, err)
				}

				var categories []string
				idToCategory := make(map[int]string)
				for _, catObj := range coco.Categories {
					categories = append(categories, catObj.Name)
					idToCategory[catObj.ID] = catObj.Name
				}

				// map from image ID to annotations in that image
				groups := make(map[int][]CocoAnnotation)
				for _, annotation := range coco.Annotations {
					groups[annotation.ImageID] = append(groups[annotation.ImageID], annotation)
				}

				for _, image := range coco.Images {
					annotations := groups[image.ID]
					var detections []skyhook.Detection
					for _, a := range annotations {
						detections = append(detections, skyhook.Detection{
							Left: int(a.Bbox[0]),
							Top: int(a.Bbox[1]),
							Right: int(a.Bbox[0]+a.Bbox[2]),
							Bottom: int(a.Bbox[1]+a.Bbox[3]),
							Category: idToCategory[a.CategoryID],
						})
					}
					outData := skyhook.DetectionData{
						Detections: [][]skyhook.Detection{detections},
						Metadata: skyhook.DetectionMetadata{
							CanvasDims: [2]int{image.Width, image.Height},
							Categories: categories,
						},
					}
					key := filenameToKey(image.Filename)
					err := exec_ops.WriteItem(url, labelDS, key, outData)
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
}
