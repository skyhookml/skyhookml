package sample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"math/rand"
	urllib "net/url"
)

type Params struct {
	// One of "count", "percentage", or "direct".
	Mode string
	// If mode=="count", the number of items to sample.
	Count int
	// If mode=="percentage", the percentage of items to sample.
	Percentage float64
	// If mode=="direct", the list of keys to sample.
	Keys []string
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "sample",
			Name: "Sample",
			Description: "Sample a subset of items from one or more datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: exec_ops.GetOutputsSimilarToInputs,
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
			// Then, sample from the tasks based on parameters.
			simpleTasks, err := exec_ops.SimpleTasks(node, allItems)
			if err != nil {
				return nil, err
			}

			var tasks []skyhook.ExecTask
			if params.Mode == "count" || params.Mode == "percentage" {
				var needed int
				if params.Mode == "count" {
					needed = params.Count
				} else if params.Mode == "percentage" {
					needed = int(float64(len(simpleTasks))*params.Percentage/100)
				}
				for _, idx := range rand.Perm(len(simpleTasks))[0:needed] {
					tasks = append(tasks, simpleTasks[idx])
				}
			} else if params.Mode == "direct" {
				keySet := make(map[string]bool)
				for _, key := range params.Keys {
					keySet[key] = true
				}
				for _, task := range simpleTasks {
					if !keySet[task.Key] {
						continue
					}
					tasks = append(tasks, task)
				}
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				// Simply copy each input to the output dataset.
				// The sampling is taken care of in GetTasks already.
				for i, itemList := range task.Items["inputs"] {
					item := itemList[0]
					dsName := fmt.Sprintf("outputs%d", i) // matches exec_ops.GetOutputsSimilarToInputs
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
