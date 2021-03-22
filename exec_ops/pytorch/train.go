package pytorch

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"os"
	"os/exec"
	"strings"
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
	var params skyhook.PytorchTrainParams
	skyhook.JsonUnmarshal([]byte(e.node.Params), &params)
	arch, components, err := GetTrainArgs(e.url, params.ArchID)
	if err != nil {
		return err
	}

	if err := EnsureRepositories(components); err != nil {
		return err
	}

	datasets, err := exec_ops.GetParentDatasets(e.url, e.node)
	if err != nil {
		return err
	}

	// run the python op
	paramsArg := e.node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasets["inputs"]))
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

// Save losses from "jsonloss" lines in the pytorch train output.
type TrainJobOp struct {
	state skyhook.ModelJobState
}
const LossSignature string = "jsonloss"
func (op *TrainJobOp) Update(lines []string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, LossSignature) {
			continue
		}
		line = line[len(LossSignature):]
		// map from train/val -> loss name -> loss value
		var data map[string]map[string]float64
		skyhook.JsonUnmarshal([]byte(line), &data)
		op.state.TrainLoss = append(op.state.TrainLoss, data["train"]["loss"])
		op.state.ValLoss = append(op.state.ValLoss, data["val"]["loss"])
	}
}
func (op *TrainJobOp) Encode() string {
	return string(skyhook.JsonMarshal(op.state))
}
func (op *TrainJobOp) Stop() error {
	// handled by ExecJobOp
	return nil
}

func init() {
	skyhook.ExecOpImpls["pytorch_train"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("model"),
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
			op := &TrainOp{
				url: url,
				node: node,
				dataset: outputDatasets["model"],
			}
			return op, nil
		},
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
		GetJobOp: func(url string, node skyhook.ExecNode) skyhook.JobOp {
			return &TrainJobOp{}
		},
	}
}
