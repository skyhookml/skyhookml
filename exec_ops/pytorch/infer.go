package pytorch

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/exec_ops/python"

	"encoding/json"
	"fmt"
	"strconv"
)

func GetInferOutputs(params skyhook.PytorchInferParams) []skyhook.ExecOutput {
	var outputs []skyhook.ExecOutput
	for i, output := range params.OutputDatasets {
		outputs = append(outputs, skyhook.ExecOutput{
			Name: fmt.Sprintf("%d-%s", i, output.Layer),
			DataType: output.DataType,
		})
	}
	return outputs
}

func Prepare(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
	// check the ArchID just to make sure we have all git repositories
	var params skyhook.PytorchInferParams
	if err := exec_ops.DecodeParams(node, &params, false); err != nil {
		return nil, err
	}
	_, components, err := GetTrainArgs(url, params.ArchID)
	if err != nil {
		return nil, err
	}
	if err := EnsureRepositories(components); err != nil {
		return nil, err
	}

	inputDatasets := node.InputDatasets

	paramsArg := node.Params
	cmd := skyhook.Command(
		fmt.Sprintf("pytorch-exec-%s", node.Name), skyhook.CommandOptions{},
		"python3", "exec_ops/pytorch/run.py",
		strconv.Itoa(inputDatasets["model"][0].ID), paramsArg,
	)

	var flatOutputs []skyhook.Dataset
	for _, output := range GetInferOutputs(params) {
		flatOutputs = append(flatOutputs, node.OutputDatasets[output.Name])
	}

	op, err := python.NewPythonOp(cmd, url, python.Params{}, inputDatasets["inputs"], flatOutputs)
	if err != nil {
		return nil, err
	}

	return op, nil
}

var InferImpl = skyhook.ExecOpImpl{
	Config: skyhook.ExecOpConfig{
		ID: "pytorch_infer",
		Name: "Pytorch (infer)",
		Description: "Pytorch (infer)",
	},
	Inputs: []skyhook.ExecInput{
		{Name: "inputs", Variable: true},
		{Name: "model", DataTypes: []skyhook.DataType{skyhook.FileType}},
	},
	GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
		var params skyhook.PytorchInferParams
		err := json.Unmarshal([]byte(rawParams), &params)
		if err != nil {
			return nil
		}
		return GetInferOutputs(params)
	},
	Requirements: func(node skyhook.Runnable) map[string]int {
		return map[string]int{"gpu": 1}
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
	ImageName: "skyhookml/pytorch",
}

func init() {
	skyhook.AddExecOpImpl(InferImpl)
}
