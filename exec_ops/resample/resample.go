package resample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	urllib "net/url"
)

func init() {
	// we use a virtual item provider for re-sampling video
	// but only when video file is unavailable
	skyhook.ItemProviders["resample_virtual"] = skyhook.VirtualProvider(func(item skyhook.Item, data skyhook.Data) (skyhook.Data, error) {
		vdata := data.(skyhook.VideoData)
		var metadata skyhook.VideoMetadata
		skyhook.JsonUnmarshal([]byte(item.Metadata), &metadata)
		vdata.Metadata = metadata
		return vdata, nil
	}, true)
}

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

			var metadata skyhook.VideoMetadata
			skyhook.JsonUnmarshal([]byte(item.Metadata), &metadata)
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
				// if no filename is available, we need to stack a virtual operation
				// on top of the input dataset
				return skyhook.JsonPostForm(e.URL, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
					"key": {task.Key},
					"ext": {item.Ext},
					"format": {item.Format},
					"metadata": {string(skyhook.JsonMarshal(metadata))},
					"provider": {"resample_virtual"},
					"provider_info": {string(skyhook.JsonMarshal(item))},
				}, nil)
			}
		}

		// re-sample by building via slice method
		// if not video, the input must be slice type
		data, err := item.LoadData()
		if err != nil {
			return err
		}
		sliceData := data.(skyhook.SliceData)

		builder := skyhook.DataImpls[data.Type()].Builder()
		outputLength := sliceData.Length() * fraction[0] / fraction[1]
		for i := 0; i < outputLength; i++ {
			idx := i*fraction[1]/fraction[0]
			builder.Write(sliceData.Slice(idx, idx+1))
		}

		outputData, err := builder.Close()
		if err != nil {
			return err
		}
		return exec_ops.WriteItem(e.URL, dataset, task.Key, outputData)
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
	skyhook.ExecOpImpls["resample"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured", err)
			}
			op := &Resample{
				URL: url,
				Params: params,
				Datasets: node.OutputDatasets,
			}
			return op, nil
		},
		GetOutputs: exec_ops.GetOutputsSimilarToInputs,
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
