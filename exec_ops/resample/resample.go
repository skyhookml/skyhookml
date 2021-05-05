package resample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"runtime"
	"strconv"
	"strings"
	urllib "net/url"
)

type Params struct {
	Fraction string
}

func (params Params) GetFraction() [2]int {
	if !strings.Contains(params.Fraction, "/") {
		x, _ := strconv.Atoi(params.Fraction)
		return [2]int{x, 1}
	}
	parts := strings.Split(params.Fraction, "/")
	numerator, _ := strconv.Atoi(parts[0])
	denominator, _ := strconv.Atoi(parts[1])
	return [2]int{numerator, denominator}
}

type Resample struct {
	URL string
	Params Params
	Datasets map[string]skyhook.Dataset
}

func (e *Resample) Parallelism() int {
	// if we resample video, each ffmpeg runs with two threads
	return runtime.NumCPU()/2
}

func (e *Resample) Apply(task skyhook.ExecTask) error {
	fraction := e.Params.GetFraction()

	process := func(item skyhook.Item, dataset skyhook.Dataset) error {
		if item.Dataset.DataType == skyhook.VideoType {
			// all we need to do is update the framerate in the metadata

			metadata := item.DecodeMetadata().(skyhook.VideoMetadata)
			metadata.Framerate = [2]int{metadata.Framerate[0]*fraction[0], metadata.Framerate[1]*fraction[1]}

			fname := item.Fname()
			if fname != "" {
				// if the filename is available, we can produce output as a reference
				// with modified metadata to the original file
				return skyhook.JsonPostForm(e.URL, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
					"key": {task.Key},
					"ext": {item.Ext},
					"format": {item.Format},
					"metadata": {string(skyhook.JsonMarshal(metadata))},
					"provider": {"reference"},
					"provider_info": {item.Fname()},
				}, nil)
			} else {
				// Filename should always be available since we shouldn't be loading video into memory.
				return fmt.Errorf("cannot resample video item that is not available on disk")
			}
		}

		// re-sample by building via Writer
		// if not video, the input must be sequence type
		data, metadata, err := item.LoadData()
		if err != nil {
			return err
		}
		spec := item.DataSpec().(skyhook.SequenceDataSpec)

		outItem, err := exec_ops.AddItem(e.URL, dataset, task.Key, item.Ext, item.Format, metadata)
		if err != nil {
			return err
		}
		writer := outItem.LoadWriter()

		outputLength := spec.Length(data) * fraction[0] / fraction[1]
		for i := 0; i < outputLength; i++ {
			idx := i*fraction[1]/fraction[0]
			writer.Write(spec.Slice(data, idx, idx+1))
		}

		return writer.Close()
	}

	for i, itemList := range task.Items["inputs"] {
		err := process(itemList[0], e.Datasets[fmt.Sprintf("outputs%d", i)])
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Resample) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "resample",
			Name: "Resample",
			Description: "Resample sequence data at a different rate",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: exec_ops.GetOutputsSimilarToInputs,
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return nil, err
			}
			op := &Resample{
				URL: url,
				Params: params,
				Datasets: node.OutputDatasets,
			}
			return op, nil
		},
		ImageName: "skyhookml/basic",
	})
}
