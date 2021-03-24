package detection_filter

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"runtime"
)

type Params struct {
	Categories []string
	Score float64
}

type DetectionFilter struct {
	URL string
	Params Params
	Dataset skyhook.Dataset

	categories map[string]bool
}

func (e *DetectionFilter) Parallelism() int {
	return runtime.NumCPU()
}

func (e *DetectionFilter) Apply(task skyhook.ExecTask) error {
	data, err := task.Items["detections"][0][0].LoadData()
	if err != nil {
		return err
	}
	detectionData := data.(skyhook.DetectionData)
	detections := detectionData.Detections
	ndetections := make([][]skyhook.Detection, len(detections))
	for i, dlist := range detections {
		ndetections[i] = []skyhook.Detection{}
		for _, d := range dlist {
			if e.Params.Score > 0 && d.Score < e.Params.Score {
				continue
			} else if len(e.categories) > 0 && !e.categories[d.Category] {
				continue
			}
			ndetections[i] = append(ndetections[i], d)
		}
	}
	outputData := skyhook.DetectionData{
		Detections: ndetections,
		Metadata: detectionData.Metadata,
	}
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, outputData)
}

func (e *DetectionFilter) Close() {}

func init() {
	skyhook.ExecOpImpls["detection_filter"] = skyhook.ExecOpImpl{
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
			var categories map[string]bool
			if len(params.Categories) > 0 {
				categories = make(map[string]bool)
				for _, category := range params.Categories {
					categories[category] = true
				}
			}
			op := &DetectionFilter{
				URL: url,
				Params: params,
				Dataset: node.OutputDatasets["detections"],
				categories: categories,
			}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
