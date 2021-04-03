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

func truthiness(data skyhook.Data) bool {
	if data.Type() == skyhook.IntType {
		for _, x := range data.(skyhook.IntData).Ints {
			if x != 0 {
				return true
			}
		}
		return false
	} else if data.Type() == skyhook.DetectionType {
		for _, dlist := range data.(skyhook.DetectionData).Detections {
			if len(dlist) > 0 {
				return true
			}
		}
		return false
	} else if data.Type() == skyhook.ShapeType {
		for _, shapes := range data.(skyhook.ShapeData).Shapes {
			if len(shapes) > 0 {
				return true
			}
		}
		return false
	} else if data.Type() == skyhook.StringType {
		for _, str := range data.(skyhook.StringData).Strings {
			if str != "" {
				return true
			}
		}
		return false
	}

	return true
}

func (e *FilterOp) Parallelism() int {
	return runtime.NumCPU()
}

func (e *FilterOp) Apply(task skyhook.ExecTask) error {
	data, err := task.Items["input"][0][0].LoadData()
	if err != nil {
		return err
	}
	if !truthiness(data) {
		return nil
	}

	mydata := skyhook.IntData{Ints: []int{1}}
	err = exec_ops.WriteItem(e.url, e.outputDatasets["output"], task.Key, mydata)
	if err != nil {
		return err
	}

	for i, itemList := range task.Items["others"] {
		item := itemList[0]
		dsName := fmt.Sprintf("others%d", i)
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
