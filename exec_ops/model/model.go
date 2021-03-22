package model

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
)

func init() {
	skyhook.ExecOpImpls["model"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
			var params skyhook.ModelExecParams
			skyhook.JsonUnmarshal([]byte(node.Params), &params)

			var trainNode skyhook.TrainNode
			err := skyhook.JsonGet(url, fmt.Sprintf("/train-nodes/%d", params.TrainNodeID), &trainNode)
			if err != nil {
				return nil, err
			}

			return skyhook.GetTrainOp(trainNode.Op).Prepare(url, trainNode, node, outputDatasets)
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			var params skyhook.ModelExecParams
			skyhook.JsonUnmarshal([]byte(node.Params), &params)

			var trainNode skyhook.TrainNode
			err := skyhook.JsonGet(url, fmt.Sprintf("/train-nodes/%d", params.TrainNodeID), &trainNode)
			if err != nil {
				return "", err
			}

			return skyhook.GetTrainOp(trainNode.Op).ImageName(url, trainNode)
		},
	}
}
