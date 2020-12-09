package python

import (
	"../../skyhook"
)

type FilterOp struct {
	node skyhook.ExecNode
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

func (e *FilterOp) Apply(key string, inputs []skyhook.Item) (map[string][]skyhook.Data, error) {
	// make sure all inputs are non-empty
	for _, item := range inputs {
		data, err := item.LoadData()
		if err != nil {
			return nil, err
		}
		if !truthiness(data) {
			return nil, nil
		}
	}

	mydata := skyhook.IntData{Ints: []int{1}}
	return map[string][]skyhook.Data{key: {mydata}}, nil
}

func (e *FilterOp) Close() {}

func init() {
	skyhook.ExecOpImpls["filter"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode) (skyhook.ExecOp, error) {
			return &FilterOp{node}, nil
		},
	}
}
