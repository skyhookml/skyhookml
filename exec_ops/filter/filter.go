package python

import (
	"../../skyhook"
	"../../exec_ops"
	"runtime"
)

type FilterOp struct {
	url string
	node skyhook.ExecNode
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
	for _, item := range task.Items {
		data, err := item.LoadData()
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
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode, items [][]skyhook.Item, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, []skyhook.ExecTask, error) {
			op := &FilterOp{
				url: url,
				node: node,
				dataset: outputDatasets[0],
			}
			tasks := exec_ops.SimpleTasks(url, node, items)
			return op, tasks, nil
		},
	}
}
