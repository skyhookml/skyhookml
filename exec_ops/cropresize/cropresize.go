package video_sample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Node skyhook.ExecNode
	Params Params
	OutputDataset skyhook.Dataset
}

func (e *CropResize) Parallelism() int {
	// each ffmpeg runs with two threads
	return runtime.NumCPU()/2
}

func (e *CropResize) Apply(task skyhook.ExecTask) error {
	data, err := task.Items["input"][0][0].LoadData()
	if err != nil {
		return err
	}
	vdata := data.(skyhook.VideoData)

	// encode the video and write the data in a separate thread
	ch := make(chan skyhook.Image)
	donech := make(chan error, 1)
	go func() {
		r, cmd := skyhook.MakeVideo(&skyhook.ChanReader{ch}, e.Params.OutputDims(), vdata.Metadata.Framerate)
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, r)
		if err != nil {
			r.Close()
			cmd.Wait()
			donech <- err
			return
		}
		r.Close()
		cmd.Wait()
		outMeta := skyhook.VideoMetadata{
			Dims: e.Params.OutputDims(),
			Framerate: vdata.Metadata.Framerate,
			Duration: vdata.Metadata.Duration,
		}
		outData := skyhook.VideoData{
			Metadata: outMeta,
			Bytes: buf.Bytes(),
		}

		err = exec_ops.WriteItem(e.URL, e.OutputDataset, task.Key, outData)
		donech <- err
	}()

	// now read the data and pass it over ch
	start := e.Params.Start
	cropDims := e.Params.CropDims
	resizeDims := e.Params.ResizeDims
	err = vdata.Iterator().Iterate(32, func(im skyhook.Image) {
		// crop and resize
		im = im.Crop(start[0], start[1], start[0]+cropDims[0], start[1]+cropDims[1])
		if resizeDims[0] > 0 {
			im = im.Resize(resizeDims[0], resizeDims[1])
		}

		ch <- im
	})
	close(ch)
	if err != nil {
		return err
	}

	// wait for encoder thread
	encodeErr := <- donech
	if encodeErr != nil {
		return encodeErr
	}

	return nil
}

func (e *CropResize) Close() {}

func init() {
	skyhook.ExecOpImpls["cropresize"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured", err)
			}
			op := &CropResize{
				URL: url,
				Node: node,
				Params: params,
				OutputDataset: outputDatasets["output"],
			}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
