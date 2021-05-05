package render

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"runtime"
	"strconv"
)

var Colors = [][3]uint8{
	[3]uint8{255, 0, 0},
	[3]uint8{0, 255, 0},
	[3]uint8{0, 0, 255},
	[3]uint8{255, 255, 0},
	[3]uint8{0, 255, 255},
	[3]uint8{255, 0, 255},
	[3]uint8{0, 51, 51},
	[3]uint8{51, 153, 153},
	[3]uint8{102, 0, 51},
	[3]uint8{102, 51, 204},
	[3]uint8{102, 153, 204},
	[3]uint8{102, 255, 204},
	[3]uint8{153, 102, 102},
	[3]uint8{204, 102, 51},
	[3]uint8{204, 255, 102},
	[3]uint8{255, 255, 204},
	[3]uint8{121, 125, 127},
	[3]uint8{69, 179, 157},
	[3]uint8{250, 215, 160},
}


type Render struct {
	URL string
	Dataset skyhook.Dataset
}

func (e *Render) Parallelism() int {
	return runtime.NumCPU()
}

func renderFrame(dtypes []skyhook.DataType, datas []interface{}, metadatas []skyhook.DataMetadata) (skyhook.Image, error) {
	var canvas skyhook.Image
	var canvases []skyhook.Image
	for i, data := range datas {
		if dtypes[i] == skyhook.ImageType || dtypes[i] == skyhook.VideoType {
			canvas = data.([]skyhook.Image)[0].Copy()
			canvases = append(canvases, canvas)
			continue
		}

		if dtypes[i] == skyhook.IntType {
			x := data.([]int)[0]
			var text string
			categories := metadatas[i].(skyhook.IntMetadata).Categories
			if x >= 0 && x < len(categories) {
				text = categories[x]
			} else {
				text = strconv.Itoa(x)
			}
			canvas.DrawText(skyhook.RichText{Text: text})
		} else if dtypes[i] == skyhook.StringType {
			text := data.([]string)[0]
			canvas.DrawText(skyhook.RichText{Text: text})
		} else if dtypes[i] == skyhook.ShapeType {
			shapes := data.([][]skyhook.Shape)[0]
			origDims := metadatas[i].(skyhook.ShapeMetadata).CanvasDims
			targetDims := [2]int{canvas.Width, canvas.Height}
			if origDims[0] == 0 {
				origDims = targetDims
			}
			for _, shape := range shapes {
				if shape.Type == "box" {
					bounds := shape.Bounds()
					canvas.DrawRectangle(
						bounds[0]*targetDims[0]/origDims[0],
						bounds[1]*targetDims[1]/origDims[1],
						bounds[2]*targetDims[0]/origDims[0],
						bounds[3]*targetDims[1]/origDims[1],
						2, [3]uint8{255, 0, 0},
					)
				} else if shape.Type == "line" {
					canvas.DrawLine(
						shape.Points[0][0]*targetDims[0]/origDims[0],
						shape.Points[0][1]*targetDims[1]/origDims[1],
						shape.Points[1][0]*targetDims[0]/origDims[0],
						shape.Points[1][1]*targetDims[1]/origDims[1],
						1, [3]uint8{255, 0, 0},
					)
				}
			}
		} else if dtypes[i] == skyhook.DetectionType {
			detections := data.([][]skyhook.Detection)[0]
			origDims := metadatas[i].(skyhook.DetectionMetadata).CanvasDims
			targetDims := [2]int{canvas.Width, canvas.Height}
			for _, d := range detections {
				if origDims[0] != 0 && origDims != targetDims {
					d = d.Rescale(origDims, targetDims)
				}
				color := Colors[d.TrackID % len(Colors)]
				canvas.DrawRectangle(d.Left, d.Top, d.Right, d.Bottom, 2, color)
			}
		}
	}

	if len(canvases) > 1 {
		// stack the canvases vertically
		var dims [2]int
		for _, im := range canvases {
			if im.Width > dims[0] {
				dims[0] = im.Width
			}
			dims[1] += im.Height
		}
		canvas = skyhook.NewImage(dims[0], dims[1])
		heightOffset := 0
		for _, im := range canvases {
			canvas.DrawImage(0, heightOffset, im)
			heightOffset += im.Height
		}
	}

	return canvas, nil
}

func (e *Render) Apply(task skyhook.ExecTask) error {
	var inputItems []skyhook.Item
	for _, itemList := range task.Items["inputs"] {
		inputItems = append(inputItems, itemList[0])
	}
	outputType := inputItems[0].Dataset.DataType

	var outputItem skyhook.Item
	// First input should be video data or image data.
	// There may be multiple video/image that we want to render.
	// But they should all be the same type (and, if video, they must have same framerates).
	// The output will have all the video/image stacked vertically.
	if outputType == skyhook.VideoType {
		// Use video metadata of all video inputs to determine the canvas dimensions.
		var dims [2]int
		var outputMetadata skyhook.VideoMetadata
		for _, item := range inputItems {
			if item.Dataset.DataType != skyhook.VideoType {
				continue
			}
			curMetadata := item.DecodeMetadata().(skyhook.VideoMetadata)
			outputMetadata = curMetadata
			curDims := curMetadata.Dims
			if curDims[0] > dims[0] {
				dims[0] = curDims[0]
			}
			dims[1] += curDims[1]
		}
		outputMetadata.Dims = dims

		var err error
		outputItem, err = exec_ops.AddItem(e.URL, e.Dataset, task.Key, inputItems[0].Ext, inputItems[0].Format, outputMetadata)
		if err != nil {
			return err
		}
	} else if outputType == skyhook.ImageType {
		var err error
		outputItem, err = exec_ops.AddItem(e.URL, e.Dataset, task.Key, inputItems[0].Ext, inputItems[0].Format, skyhook.NoMetadata{})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("first item must be either video or image type")
	}

	writer := outputItem.LoadWriter()

	var dtypes []skyhook.DataType
	var metadatas []skyhook.DataMetadata
	for _, item := range inputItems {
		dtypes = append(dtypes, item.Dataset.DataType)
		metadatas = append(metadatas, item.DecodeMetadata())
	}
	err := skyhook.PerFrame(inputItems, func(pos int, datas []interface{}) error {
		im, err := renderFrame(dtypes, datas, metadatas)
		if err != nil {
			return err
		}
		err = writer.Write([]skyhook.Image{im})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return writer.Close()
}

func (e *Render) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "render",
			Name: "Render Images or Video",
			Description: "Render images/video from various input data types",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: func(params string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			// whether we output video or image depends on the first input
			var dtype skyhook.DataType = skyhook.VideoType
			if len(inputTypes["inputs"]) > 0 {
				dtype = inputTypes["inputs"][0]
			}
			return []skyhook.ExecOutput{{
				Name: "output",
				DataType: dtype,
			}}
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			op := &Render{url, node.OutputDatasets["output"]}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: "skyhookml/basic",
	})
}
