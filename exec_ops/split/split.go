package split

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"math/rand"
	urllib "net/url"
	"strconv"
)

type Split struct {
	Name string
	Percentage int
}

// Get the name that should be used for output of this split
// at a certain index in inputs.
func (s Split) GetOutputName(inputIdx int) string {
	if inputIdx == 0 {
		return s.Name
	} else {
		return fmt.Sprintf("%d-%s", inputIdx, s.Name)
	}
}

type Params struct {
	Splits []Split
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "split",
			Name: "Split",
			Description: "Split items in an input dataset into two or more output datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			var params Params
			err := json.Unmarshal([]byte(rawParams), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				return nil
			}

			var outputs []skyhook.ExecOutput
			for inputIdx, dtype := range inputTypes["inputs"] {
				for _, split := range params.Splits {
					outputs = append(outputs, skyhook.ExecOutput{
						Name: split.GetOutputName(inputIdx),
						DataType: dtype,
					})
				}
			}
			return outputs
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured: %v", err)
			}

			// First, get tasks for keys that appear in all datasets using SimpleTasks.
			// Then split the tasks, and encode the split index in the task metadata.
			simpleTasks, err := exec_ops.SimpleTasks(node, allItems)
			if err != nil {
				return nil, err
			}
			rand.Shuffle(len(simpleTasks), func(i, j int) {
				simpleTasks[i], simpleTasks[j] = simpleTasks[j], simpleTasks[i]
			})

			taskSplits := make([][]skyhook.ExecTask, len(params.Splits))
			var percentageSum int = 0
			var prevIdx int = 0
			for splitIdx, split := range params.Splits {
				percentageSum += split.Percentage
				nextIdx := percentageSum * len(simpleTasks) / 100
				taskSplits[splitIdx] = simpleTasks[prevIdx:nextIdx]
				prevIdx = nextIdx
			}

			var tasks []skyhook.ExecTask
			for i := range params.Splits {
				for _, task := range taskSplits[i] {
					task.Metadata = strconv.Itoa(i)
					tasks = append(tasks, task)
				}
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured: %v", err)
			}

			applyFunc := func(task skyhook.ExecTask) error {
				splitIdx := skyhook.ParseInt(task.Metadata)

				// Simply copy each input to the output dataset.
				// The splitting is taken care of in GetTasks already.
				for i, itemList := range task.Items["inputs"] {
					item := itemList[0]
					dsName := params.Splits[splitIdx].GetOutputName(i)
					err := skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", node.OutputDatasets[dsName].ID), urllib.Values{
						"key": {task.Key},
						"ext": {item.Ext},
						"format": {item.Format},
						"metadata": {item.Metadata},
						"provider": {"reference"},
						"provider_info": {item.Fname()},
					}, nil)
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
