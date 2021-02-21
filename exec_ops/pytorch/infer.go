package pytorch

import (
	"../../skyhook"
	"../../exec_ops"
	"../../exec_ops/python"

	"fmt"
)

func Prepare(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
	arch, components, _, err := GetArgs(url, node)
	if err != nil {
		return nil, err
	}

	if err := EnsureRepositories(components); err != nil {
		return nil, err
	}

	// get the model path from the first input dataset
	datasets, err := exec_ops.ParentsToDatasets(url, node.Parents[0:1])
	if err != nil {
		return nil, err
	}
	modelItems, err := exec_ops.GetItems(url, datasets)
	if err != nil {
		return nil, err
	}
	modelItem := modelItems["model"][0]
	strdata, err := modelItem.LoadData()
	if err != nil {
		return nil, err
	}
	modelPath := strdata.(skyhook.StringData).Strings[0]

	paramsArg := node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	cmd := skyhook.Command(
		fmt.Sprintf("pytorch-exec-%s", node.Name), skyhook.CommandOptions{},
		"python3", "exec_ops/pytorch/run.py",
		modelPath, paramsArg, archArg, compsArg,
	)

	op, err := python.NewPythonOp(cmd, url, node, outputDatasets)
	return op, err
}

func init() {
	skyhook.ExecOpImpls["pytorch_infer"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: func(url string, node skyhook.ExecNode, rawItems [][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// the first input dataset in the model
			// so we just provide the rest to SimpleTasks
			return exec_ops.SimpleTasks(url, node, rawItems[1:])
		},
		Prepare: Prepare,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
