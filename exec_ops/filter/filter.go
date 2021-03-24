package python

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"runtime"
)

type FilterOp struct {
	url string
	dataset skyhook.Dataset
}

func truthiness(data skyhook.Data) bool {
	if data.Type() == skyhook.IntType {
		any := false
		for _, x := range data.(skyhook.IntData).Ints {
			if x != 0 {
				any = true
			}
		}
		return any
	}
	return false
}

func (e *FilterOp) Parallelism() int {
	return runtime.NumCPU()
}

func (e *FilterOp) Apply(task skyhook.ExecTask) error {
	// make sure all inputs are non-empty
	for _, itemList := range task.Items["inputs"] {
		data, err := itemList[0].LoadData()
		if err != nil {
			return err
		}
		if !truthiness(data) {
			return nil
		}
	}

	mydata := skyhook.IntData{Ints: []int{1}}
	return exec_ops.WriteItem(e.url, e.dataset, task.Key, mydata)
}

func (e *FilterOp) Close() {}

func init() {
	skyhook.ExecOpImpls["filter"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			op := &FilterOp{
				url: url,
				dataset: node.OutputDatasets["output"],
			}
			return op, nil
		},
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
