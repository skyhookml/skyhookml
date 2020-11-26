package model

import (
	"../../skyhook"

	"fmt"
)

func init() {
	skyhook.ExecOpImpls["model"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode) (skyhook.ExecOp, error) {
			var params skyhook.ModelExecParams
			skyhook.JsonUnmarshal([]byte(node.Params), &params)

			var trainNode skyhook.TrainNode
			err := skyhook.JsonGet(url, fmt.Sprintf("/train-nodes/%d", params.TrainNodeID), &trainNode)
			if err != nil {
				return nil, err
			}

			return skyhook.GetTrainOp(trainNode.Op).Prepare(url, trainNode, node)
		},
	}
}
