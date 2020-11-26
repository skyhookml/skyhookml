package keras

import (
	"../../skyhook"
	"../../exec_ops/python"

	"fmt"
	"os"
	"os/exec"
)

type KerasNodeParams struct {
	Archs []struct{
		ID int
		// Sources for the arch inputs.
		Inputs []string
		// List of layer names in this arch that get added to outputs.
		Outputs []string
	}
	LoadFrom []int
	TrainLayers []string
	InputDatasets []int
	OutputDatasets []int
}

func Train(url string, node skyhook.TrainNode) error {
	var params KerasNodeParams
	skyhook.JsonUnmarshal([]byte(node.Params), &params)

	// get the KerasArchs
	archs := make([]skyhook.KerasArch, len(params.Archs))
	for i, x := range params.Archs {
		var arch skyhook.KerasArch
		err := skyhook.JsonGet(url, fmt.Sprintf("/keras/archs/%d", x.ID), &arch)
		if err != nil {
			return err
		}
		archs[i] = arch
	}

	// get the Datasets
	datasets := make(map[int]*skyhook.Dataset)
	var dsIDs []int
	dsIDs = append(dsIDs, params.InputDatasets...)
	dsIDs = append(dsIDs, params.OutputDatasets...)
	for _, dsID := range dsIDs {
		if datasets[dsID] != nil {
			continue
		}
		var ds *skyhook.Dataset
		err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", dsID), &ds)
		if err != nil {
			return err
		}
		datasets[dsID] = ds
	}

	// run the python op
	paramsArg := node.Params
	archsArg := string(skyhook.JsonMarshal(archs))
	datasetsArg := string(skyhook.JsonMarshal(datasets))
	fmt.Println(node.ID, paramsArg, archsArg, datasetsArg)
	cmd := exec.Command(
		"python3", "train_ops/keras/train.py",
		fmt.Sprintf("%d", node.ID), url, paramsArg, archsArg, datasetsArg,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	return err
}

func Prepare(url string, trainNode skyhook.TrainNode, execNode skyhook.ExecNode) (skyhook.ExecOp, error) {
	var params KerasNodeParams
	skyhook.JsonUnmarshal([]byte(trainNode.Params), &params)

	// get the KerasArchs
	archs := make([]skyhook.KerasArch, len(params.Archs))
	for i, x := range params.Archs {
		var arch skyhook.KerasArch
		err := skyhook.JsonGet(url, fmt.Sprintf("/keras/archs/%d", x.ID), &arch)
		if err != nil {
			return nil, err
		}
		archs[i] = arch
	}

	paramsArg := trainNode.Params
	archsArg := string(skyhook.JsonMarshal(archs))
	execParamsArg := execNode.Params
	cmd := skyhook.Command(
		fmt.Sprintf("kerasexec-%s", execNode.Name), skyhook.CommandOptions{},
		"python3", "train_ops/keras/run.py",
		fmt.Sprintf("%d", trainNode.ID), paramsArg, archsArg, execParamsArg,
	)

	op, err := python.NewPythonOp(cmd, url, execNode)
	return op, err
}

func init() {
	skyhook.TrainOps["keras"] = skyhook.TrainOp{
		Requirements: func(url string, node skyhook.TrainNode) map[string]int {
			return map[string]int{}
		},
		Train: Train,
		Prepare: Prepare,
	}
}
