package mask

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"runtime"
)

type Params struct {
	Dims [2]int
}

type Mask struct {
	Params Params
	URL string
	OutputDataset skyhook.Dataset
}

func (e *Mask) Parallelism() int {
	return runtime.NumCPU()
}

// TODO: handle numCategories>256
func (e *Mask) renderFrame(data skyhook.Data, categoryMap map[string]int) ([]byte, error) {
	dims := e.Params.Dims
	canvas := make([]byte, dims[0]*dims[1])

	fillRectangle := func(sx, sy, ex, ey, cls int) {
		sx = skyhook.Clip(sx, 0, dims[0])
		sy = skyhook.Clip(sy, 0, dims[1])
		ex = skyhook.Clip(ex, 0, dims[0])
		ey = skyhook.Clip(ey, 0, dims[1])
		for x := sx; x < ex; x++ {
			for y := sy; y < ey; y++ {
				canvas[y*dims[0] + x] = byte(cls)
			}
		}
	}

	// category string to ID
	getCategoryID := func(name string) int {
		if categoryMap[name] != 0 {
			return categoryMap[name]
		}

		// looks like the category string is not in the category list
		// if we are creating a two-category output, then that's okay, we can just set it to 1
		// otherwise we should return an error
		if len(categoryMap) == 1 {
			return 1
		}
		return -1
	}

	if data.Type() == skyhook.ShapeType {
		shapeData := data.(skyhook.ShapeData)
		shapes := shapeData.Shapes[0]
		shapeDims := shapeData.Metadata.CanvasDims
		if shapeDims[0] == 0 {
			// if no dims set in data, assume it corresponds to output dims
			shapeDims = dims
		}
		for _, shape := range shapes {
			if shape.Type == "box" {
				bounds := shape.Bounds()
				catID := getCategoryID(shape.Category)
				if catID == -1 {
					return nil, fmt.Errorf("unknown category %s", shape.Category)
				}
				fillRectangle(
					bounds[0]*dims[0]/shapeDims[0],
					bounds[1]*dims[1]/shapeDims[1],
					bounds[2]*dims[0]/shapeDims[0],
					bounds[3]*dims[1]/shapeDims[1],
					catID,
				)
			} else if shape.Type == "line" {
				panic(fmt.Errorf("line shape mask not implemented"))
			}
		}
	} else if data.Type() == skyhook.DetectionType {
		detectionData := data.(skyhook.DetectionData)
		detections := detectionData.Detections[0]
		detDims := detectionData.Metadata.CanvasDims
		for _, d := range detections {
			if detDims[0] != 0 && detDims != dims {
				d = d.Rescale(detDims, dims)
			}
			catID := getCategoryID(d.Category)
			if catID == -1 {
				return nil, fmt.Errorf("unknown category %s", d.Category)
			}
			fillRectangle(d.Left, d.Top, d.Right, d.Bottom, catID)
		}
	}

	return canvas, nil
}

func (e *Mask) Apply(task skyhook.ExecTask) error {
	input, err := task.Items["input"][0][0].LoadData()
	if err != nil {
		return err
	}

	var categories []string
	if input.Type() == skyhook.ShapeType {
		categories = input.(skyhook.ShapeData).Metadata.Categories
	} else if input.Type() == skyhook.DetectionType {
		categories = input.(skyhook.DetectionData).Metadata.Categories
	}

	numCategories := len(categories)+1
	categoryMap := make(map[string]int)
	if numCategories == 1 {
		// input dataset doesn't have category labels, so we assume user just wants background/foreground
		// then, anything in the input dataset should be foreground
		numCategories = 2
		categoryMap[""] = 1
	} else {
		for i, category := range categories {
			// 0 is reserved for "background"
			categoryMap[category] = i+1
		}
	}

	metadata := skyhook.ArrayMetadata{
		Width: e.Params.Dims[0],
		Height: e.Params.Dims[1],
		Channels: 1,
		Type: "uint8",
	}

	builder := skyhook.DataImpls[skyhook.ArrayType].Builder()
	skyhook.PerFrame([]skyhook.Data{input}, func(pos int, datas []skyhook.Data) error {
		frameBytes, err := e.renderFrame(datas[0], categoryMap)
		if err != nil {
			return err
		}
		return builder.Write(skyhook.ArrayData{
			Bytes: frameBytes,
			Metadata: metadata,
		})
	})

	output, err := builder.Close()
	if err != nil {
		return err
	}
	return exec_ops.WriteItem(e.URL, e.OutputDataset, task.Key, output)
}

func (e *Mask) Close() {}

func init() {
	skyhook.ExecOpImpls["segmentation_mask"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				return nil, fmt.Errorf("node has not been configured")
			}
			return &Mask{
				Params: params,
				URL: url,
				OutputDataset: node.OutputDatasets["output"],
			}, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
