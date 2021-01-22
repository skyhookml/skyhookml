package keras

import (
	"../../skyhook"
	"../../exec_ops/python"

	"fmt"
	"os"
	"os/exec"
)

func getArgs(url string, node skyhook.TrainNode) (*skyhook.PytorchArch, map[int]*skyhook.PytorchComponent, map[int]*skyhook.Dataset, error) {
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

func Train(url string, node skyhook.TrainNode) error {
	arch, components, datasets, err := getArgs(url, node)
	if err != nil {
		return err
	}

	// run the python op
	paramsArg := node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasets))
	fmt.Println(node.ID, paramsArg, archArg, compsArg, datasetsArg)
	cmd := exec.Command(
		"python3", "train_ops/pytorch/train.py",
		fmt.Sprintf("%d", node.ID), url, paramsArg, archArg, compsArg, datasetsArg,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func Prepare(url string, trainNode skyhook.TrainNode, execNode skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
	arch, components, _, err := getArgs(url, trainNode)
	if err != nil {
		return nil, err
	}

	paramsArg := trainNode.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	execParamsArg := execNode.Params
	cmd := skyhook.Command(
		fmt.Sprintf("pytorch-exec-%s", execNode.Name), skyhook.CommandOptions{},
		"python3", "train_ops/pytorch/run.py",
		fmt.Sprintf("%d", trainNode.ID), paramsArg, archArg, compsArg, execParamsArg,
	)

	op, err := python.NewPythonOp(cmd, url, execNode, outputDatasets)
	return op, err
}

func init() {
	skyhook.TrainOps["pytorch"] = skyhook.TrainOp{
		Requirements: func(url string, node skyhook.TrainNode) map[string]int {
			return map[string]int{}
		},
		Train: Train,
		Prepare: Prepare,
		ImageName: func(url string, node skyhook.TrainNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
