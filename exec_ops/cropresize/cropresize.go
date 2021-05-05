package video_sample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"runtime"
)

// TODO:
// - support image inputs
// - build window annotation tool which can annotate either:
//   - one window for entire dataset
//   - or one window per item in dataset (but not per frame)
// - and also should be able to call that annotation tool without creating a new dataset
//   - like if we just want to annotate a window directly from the cropresize node editor
// might want coordinates and dimensions to be expressed as fractions so they can work on different sized inputs

type Params struct {
	Start [2]int
	CropDims [2]int
	ResizeDims [2]int
}

func (params Params) OutputDims() [2]int {
	if params.ResizeDims[0] > 0 {
		return params.ResizeDims
	} else {
		return params.CropDims
	}
}

type CropResize struct {
	URL string
	Params Params
	OutputDataset skyhook.Dataset
}

func (e *CropResize) Parallelism() int {
	// each ffmpeg runs with two threads
	return runtime.NumCPU()/2
}

func (e *CropResize) Apply(task skyhook.ExecTask) error {
	inputItem := task.Items["input"][0][0]
	inputMetadata := inputItem.DecodeMetadata().(skyhook.VideoMetadata)

	// Initialize writer.
	outputMetadata := skyhook.VideoMetadata{
		Dims: e.Params.OutputDims(),
		Framerate: inputMetadata.Framerate,
		Duration: inputMetadata.Duration,
	}
	outputItem, err := exec_ops.AddItem(e.URL, e.OutputDataset, task.Key, inputItem.Ext, inputItem.Format, outputMetadata)
	if err != nil {
		return err
	}
	writer := outputItem.LoadWriter()

	// Use PerFrame to get input frames.
	start := e.Params.Start
	cropDims := e.Params.CropDims
	resizeDims := e.Params.ResizeDims
	err = skyhook.PerFrame([]skyhook.Item{inputItem}, func(pos int, datas []interface{}) error {
		im := datas[0].([]skyhook.Image)[0]

		// Crop and resize.
		im = im.Crop(start[0], start[1], start[0]+cropDims[0], start[1]+cropDims[1])
		if resizeDims[0] > 0 {
			im = im.Resize(resizeDims[0], resizeDims[1])
		}

		return writer.Write([]skyhook.Image{im})
	})
	if err != nil {
		return err
	}

	return writer.Close()
}

func (e *CropResize) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "cropresize",
			Name: "Crop/Resize Video",
			Description: "Crop video followed by optional resize",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.VideoType}}},
		Outputs: []skyhook.ExecOutput{{Name: "output", DataType: skyhook.VideoType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return nil, err
			}
			op := &CropResize{
				URL: url,
				Params: params,
				OutputDataset: node.OutputDatasets["output"],
			}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: "skyhookml/basic",
	})
}
