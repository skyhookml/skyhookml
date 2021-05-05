package python

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	urllib "net/url"
	"runtime"
)

type FilterOp struct {
	url string
	outputDatasets map[string]skyhook.Dataset
}

var truthinessMap = map[skyhook.DataType]func(data interface{}) bool{
	skyhook.IntType: func(data interface{}) bool {
		for _, x := range data.([]int) {
			if x != 0 {
				return true
			}
		}
		return false
	},
	skyhook.DetectionType: func(data interface{}) bool {
		for _, dlist := range data.([][]skyhook.Detection) {
			if len(dlist) > 0 {
				return true
			}
		}
		return false
	},
	skyhook.ShapeType: func(data interface{}) bool {
		for _, shapes := range data.([][]skyhook.Shape) {
			if len(shapes) > 0 {
				return true
			}
		}
		return false
	},
	skyhook.StringType: func(data interface{}) bool {
		for _, str := range data.([]string) {
			if str != "" {
				return true
			}
		}
		return false
	},
}

func truthiness(item skyhook.Item) (bool, error) {
	f, ok := truthinessMap[item.Dataset.DataType]
	if !ok {
		// Other types are always considered truthy.
		return true, nil
	}
	data, _, err := item.LoadData()
	if err != nil {
		return false, err
	}
	return f(data), nil
}

func (e *FilterOp) Parallelism() int {
	return runtime.NumCPU()
}

func (e *FilterOp) Apply(task skyhook.ExecTask) error {
	truthy, err := truthiness(task.Items["input"][0][0])
	if err != nil {
		return err
	} else if !truthy {
		return nil
	}

	mydata := []int{1}
	err = exec_ops.WriteItem(e.url, e.outputDatasets["output"], task.Key, mydata, skyhook.IntMetadata{})
	if err != nil {
		return err
	}

	for i, itemList := range task.Items["others"] {
		item := itemList[0]
		dsName := fmt.Sprintf("others%d", i)
		// TODO: make this work even if Fname() isn't available.
		err := skyhook.JsonPostForm(e.url, fmt.Sprintf("/datasets/%d/items", e.outputDatasets[dsName].ID), urllib.Values{
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

func (e *FilterOp) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "filter",
			Name: "Filter",
			Description: "Filter",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "input"},
			{Name: "others", Variable: true},
		},
		GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			// output is always int type
			// but others copies the type of each input in others
			outputs := []skyhook.ExecOutput{{
				Name: "output",
				DataType: skyhook.IntType,
			}}
			for i, inputType := range inputTypes["others"] {
				outputs = append(outputs, skyhook.ExecOutput{
					Name: fmt.Sprintf("others%d", i),
					DataType: inputType,
				})
			}
			return outputs
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// use "input" to determine which tasks to create
			otherItemsByKey := make(map[string][][]skyhook.Item)
			for i, itemList := range allItems["others"] {
				for _, item := range itemList {
					key := item.Key
					if otherItemsByKey[key] == nil {
						otherItemsByKey[key] = make([][]skyhook.Item, len(allItems["others"]))
					}
					otherItemsByKey[key][i] = []skyhook.Item{item}
				}
			}
			var tasks []skyhook.ExecTask
			for _, item := range allItems["input"][0] {
				tasks = append(tasks, skyhook.ExecTask{
					Key: item.Key,
					Items: map[string][][]skyhook.Item{
						"input": {{item}},
						"others": otherItemsByKey[item.Key],
					},
				})
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			op := &FilterOp{
				url: url,
				outputDatasets: node.OutputDatasets,
			}
			return op, nil
		},
		ImageName: "skyhookml/basic",
	})
}
