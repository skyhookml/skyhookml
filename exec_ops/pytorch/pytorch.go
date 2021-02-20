package pytorch

import (
	"../../skyhook"

	"fmt"
)

func GetArgs(url string, node skyhook.ExecNode) (*skyhook.PytorchArch, map[int]*skyhook.PytorchComponent, map[int]*skyhook.Dataset, error) {
	var params skyhook.PytorchNodeParams
	skyhook.JsonUnmarshal([]byte(node.Params), &params)

	// get the PytorchComponents
	var arch skyhook.PytorchArch
	err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/archs/%d", params.ArchID), &arch)
	if err != nil {
		return nil, nil, nil, err
	}
	components := make(map[int]*skyhook.PytorchComponent)
	for _, compSpec := range arch.Params.Components {
		if components[compSpec.ID] != nil {
			continue
		}
		var comp skyhook.PytorchComponent
		err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/components/%d", compSpec.ID), &comp)
		if err != nil {
			return nil, nil, nil, err
		}
		components[comp.ID] = &comp
	}

	// get the Datasets
	datasets := make(map[int]*skyhook.Dataset)
	for _, dsSpec := range params.InputDatasets {
		if datasets[dsSpec.ID] != nil {
			continue
		}
		var ds skyhook.Dataset
		err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", dsSpec.ID), &ds)
		if err != nil {
			return nil, nil, nil, err
		}
		datasets[dsSpec.ID] = &ds
	}

	return &arch, components, datasets, nil
}
