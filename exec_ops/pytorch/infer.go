package pytorch

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/exec_ops/python"

	"encoding/json"
	"fmt"
)

func Prepare(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
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

	inputDatasets := node.InputDatasets

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
		flatOutputs = append(flatOutputs, node.OutputDatasets[output.Name])
	}

	return python.NewPythonOp(cmd, url, node, inputDatasets["inputs"], flatOutputs)
}

func init() {
	skyhook.ExecOpImpls["pytorch_infer"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// the model only has one dataset, we want to use all the other datasets
			// should just be under "inputs"
			items := make(map[string][][]skyhook.Item)
			for name, value := range rawItems {
				if name == "model" {
					continue
				}
				items[name] = value
			}
			return exec_ops.SimpleTasks(node, items)
		},
		Prepare: Prepare,
		GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			var params skyhook.PytorchInferParams
			err := json.Unmarshal([]byte(rawParams), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				// so we leave it unchanged
				return nil
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
		Incremental: true,
		GetOutputKeys: func(node skyhook.ExecNode, inputs map[string][][]string) []string {
			inputsWithoutModel := make(map[string][][]string)
			for name, value := range inputs {
				if name == "model" {
					continue
				}
				inputsWithoutModel[name] = value
			}
			return exec_ops.MapGetOutputKeys(node, inputsWithoutModel)
		},
		GetNeededInputs: func(node skyhook.ExecNode, outputs []string) map[string][][]string {
			neededInputs := exec_ops.MapGetNeededInputs(node, outputs)
			neededInputs["model"] = [][]string{{"model"}}
			return neededInputs
		},
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
