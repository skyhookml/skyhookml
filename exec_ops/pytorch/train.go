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
	node skyhook.Runnable
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

	datasets := e.node.InputDatasets

	// run the python op
	paramsArg := e.node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasets["inputs"]))
	fmt.Println(e.dataset.ID, paramsArg, archArg, compsArg, datasetsArg)
	cmd := exec.Command(
		"python3", "exec_ops/pytorch/train.py",
		fmt.Sprintf("%d", e.dataset.ID), e.url, paramsArg, archArg, compsArg, datasetsArg,
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
	mydata := skyhook.StringData{Strings: []string{fmt.Sprintf("%d", e.dataset.ID)}}
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
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("model"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			op := &TrainOp{
				url: url,
				node: node,
				dataset: node.OutputDatasets["model"],
			}
			return op, nil
		},
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/pytorch", nil
		},
		GetJobOp: func(node skyhook.Runnable) skyhook.JobOp {
			return &TrainJobOp{}
		},
		Resolve: func(node *skyhook.VirtualNode, inputDatasets map[string][]skyhook.Dataset, items map[string][][]skyhook.Item) skyhook.ExecutionGraph {
			// If parent items include non-materialized data (non-default provider),
			// then we need to run materialize op on those datasets.

			// list of names and indices that need materialization
			type ParentSpec struct {
				Name string
				Index int
			}
			var needed []ParentSpec
			for name, itemLists := range items {
				for idx, itemList := range itemLists {
					ok := true
					for _, item := range itemList {
						if item.Provider != nil {
							ok = false
							break
						}
					}
					if ok {
						continue
					}
					needed = append(needed, ParentSpec{
						Name: name,
						Index: idx,
					})
				}
			}

			if len(needed) == 0 {
				return nil
			}

			subgraph := make(skyhook.ExecutionGraph)
			origGID := node.GraphID()

			// create a materialize node to materialize the needed ones
			var matParents []skyhook.VirtualParent
			var matInputTypes []skyhook.DataType
			specToMatOutputIndex := make(map[ParentSpec]int)
			for i, spec := range needed {
				matParents = append(matParents, node.GetParents()[spec.Name][spec.Index])
				matInputTypes = append(matInputTypes, inputDatasets[spec.Name][spec.Index].DataType)
				specToMatOutputIndex[spec] = i
			}
			matGID := skyhook.GraphID{
				Type: origGID.Type,
				ID: origGID.ID,
				VirtualKey: origGID.VirtualKey+"/materialize",
			}
			subgraph[matGID] = &skyhook.VirtualNode{
				Name: node.Name+"-materialize",
				Op: "materialize",
				Params: "",
				Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
				Outputs: skyhook.GetExecOpImpl("materialize").GetOutputs("", map[string][]skyhook.DataType{"inputs": matInputTypes}),
				Parents: [][]skyhook.VirtualParent{matParents},
				OrigNode: node.OrigNode,
				VirtualKey: matGID.VirtualKey,
			}

			// and we need to update the pytorch node to input from the materialize node
			for i := range node.Parents {
				name := node.Inputs[i].Name
				for idx := range node.Parents[i] {
					matOutputIndex, ok := specToMatOutputIndex[ParentSpec{name, idx}]
					if !ok {
						continue
					}
					node.Parents[i][idx] = skyhook.VirtualParent{
						GraphID: matGID,
						Name: fmt.Sprintf("outputs%d", matOutputIndex),
					}
				}
			}
			subgraph[origGID] = node

			return subgraph
		},
	}
}
