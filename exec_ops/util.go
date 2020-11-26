package exec_ops

import (
	"../skyhook"

	"fmt"
)

func GetParentDatasets(url string, node skyhook.ExecNode) ([]skyhook.Dataset, error) {
	datasets := make([]skyhook.Dataset, len(node.Parents))
	for i, parent := range node.Parents {
		if parent.Type == "n" {
			var parentNode skyhook.ExecNode
			err := skyhook.JsonGet(url, fmt.Sprintf("/exec-nodes/%d", parent.ID), &parentNode)
			if err != nil {
				return nil, fmt.Errorf("error getting parent node %d: %v", parent.ID, err)
			}
			dsID := parentNode.DatasetIDs[parent.Index]
			if dsID == nil {
				return nil, fmt.Errorf("parent %s missing dataset at index %d", parentNode.Name, parent.Index)
			}

			var dataset skyhook.Dataset
			err = skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", *dsID), &dataset)
			if err != nil {
				return nil, fmt.Errorf("error getting dataset for parent node %s: %v", parentNode.Name, err)
			}
			datasets[i] = dataset
		} else if parent.Type == "d" {
			var dataset skyhook.Dataset
			err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", parent.ID), &dataset)
			if err != nil {
				return nil, fmt.Errorf("error getting parent dataset %d: %v", parent.ID, err)
			}
			datasets[i] = dataset
		}
	}
	return datasets, nil
}
