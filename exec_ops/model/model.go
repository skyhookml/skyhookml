package model

import (
	"../../skyhook"
	"../../exec_ops"

	"fmt"
)

func init() {
	skyhook.ExecOpImpls["model"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode, items [][]skyhook.Item, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, []skyhook.ExecTask, error) {
			var params skyhook.ModelExecParams
			skyhook.JsonUnmarshal([]byte(node.Params), &params)

			var trainNode skyhook.TrainNode
			err := skyhook.JsonGet(url, fmt.Sprintf("/train-nodes/%d", params.TrainNodeID), &trainNode)
			if err != nil {
				return nil, nil, err
			}

			return skyhook.GetTrainOp(trainNode.Op).Prepare(url, trainNode, node, items, outputDatasets)
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
	}
}
