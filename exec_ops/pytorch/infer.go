package pytorch

import (
	"../../skyhook"
	"../../exec_ops"
	"../../exec_ops/python"

	"encoding/json"
	"fmt"
)

func Prepare(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
	// check the ArchID just to make sure we have all git repositories
	var params skyhook.PytorchInferParams
	skyhook.JsonUnmarshal([]byte(node.Params), &params)
	_, components, err := GetTrainArgs(url, params.ArchID)
	if err != nil {
		return nil, err
	}
	if err := EnsureRepositories(components); err != nil {
		return nil, err
	}

	inputDatasets, err := exec_ops.ParentsToDatasets(url, node.GetParents())
	if err != nil {
		return nil, err
	}

	// get the model path from the model input dataset
	modelItems, err := exec_ops.GetDatasetItems(url, inputDatasets["model"][0])
	if err != nil {
		return nil, err
	}
	strdata, err := modelItems["model"].LoadData()
	if err != nil {
		return nil, err
	}
	modelPath := strdata.(skyhook.StringData).Strings[0]

	paramsArg := node.Params
	cmd := skyhook.Command(
		fmt.Sprintf("pytorch-exec-%s", node.Name), skyhook.CommandOptions{},
		"python3", "exec_ops/pytorch/run.py",
		modelPath, paramsArg,
	)

	var flatOutputs []skyhook.Dataset
	for _, output := range node.Outputs {
		flatOutputs = append(flatOutputs, outputDatasets[output.Name])
	}

	return python.NewPythonOp(cmd, url, node, inputDatasets["inputs"], flatOutputs)
}

func init() {
	skyhook.ExecOpImpls["pytorch_infer"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: func(url string, node skyhook.ExecNode, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// the model only has one dataset, we want to use all the other datasets
			// should just be under "inputs"
			items := make(map[string][][]skyhook.Item)
			for name, value := range rawItems {
				if name == "model" {
					continue
				}
				items[name] = value
			}
			return exec_ops.SimpleTasks(url, node, items)
		},
		Prepare: Prepare,
		GetOutputs: func(url string, node skyhook.ExecNode) []skyhook.ExecOutput {
			var params skyhook.PytorchInferParams
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				// so we leave it unchanged
				return node.Outputs
			}

			var outputs []skyhook.ExecOutput
			for i, output := range params.OutputDatasets {
				outputs = append(outputs, skyhook.ExecOutput{
					Name: fmt.Sprintf("%d-%s", i, output.Layer),
					DataType: output.DataType,
				})
			}
			return outputs
		},
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
