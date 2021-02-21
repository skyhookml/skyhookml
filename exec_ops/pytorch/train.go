package pytorch

import (
	"../../skyhook"
	"../../exec_ops"

	"fmt"
	"os"
	"os/exec"
)

type TrainOp struct {
	url string
	node skyhook.ExecNode
	dataset skyhook.Dataset
}

func (e *TrainOp) Parallelism() int {
	return 1
}

func (e *TrainOp) Apply(task skyhook.ExecTask) error {
	arch, components, datasets, err := GetArgs(e.url, e.node)
	if err != nil {
		return err
	}

	if err := EnsureRepositories(components); err != nil {
		return err
	}

	// run the python op
	paramsArg := e.node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasets))
	fmt.Println(e.node.ID, paramsArg, archArg, compsArg, datasetsArg)
	cmd := exec.Command(
		"python3", "exec_ops/pytorch/train.py",
		fmt.Sprintf("%d", e.node.ID), e.url, paramsArg, archArg, compsArg, datasetsArg,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	// add filename to the string dataset
	mydata := skyhook.StringData{Strings: []string{fmt.Sprintf("%d", e.node.ID)}}
	return exec_ops.WriteItem(e.url, e.dataset, "model", mydata)
}

func (e *TrainOp) Close() {}

func init() {
	skyhook.ExecOpImpls["pytorch_train"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("model"),
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
			op := &TrainOp{
				url: url,
				node: node,
				dataset: outputDatasets[0],
			}
			return op, nil
		},
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
