package union

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"strconv"
	urllib "net/url"
)

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "union",
			Name: "Union",
			Description: "Create an output dataset that includes all items from all input datasets",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: func(params string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			if len(inputTypes["inputs"]) == 0 {
				return nil
			}
			return []skyhook.ExecOutput{{Name: "output", DataType: inputTypes["inputs"][0]}}
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// Create one task per item.
			// We set the Key of the task to the output key that we want to write as.
			// This may not match the input key in the case that there are duplicate keys across input datasets.
			seenKeys := make(map[string]bool)
			var tasks []skyhook.ExecTask
			for _, itemList := range rawItems["inputs"] {
				for _, item := range itemList {
					key := item.Key
					for i := 1; seenKeys[key]; i++ {
						key = item.Key + "-" + strconv.Itoa(i)
					}
					seenKeys[key] = true
					tasks = append(tasks, skyhook.ExecTask{
						Key: key,
						Items: map[string][][]skyhook.Item{"inputs": {{item}}},
					})
				}
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				item := task.Items["inputs"][0][0]
				outDataset := node.OutputDatasets["output"]
				err := skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", outDataset.ID), urllib.Values{
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
				return nil
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
