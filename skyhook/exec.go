package skyhook

import (
	"strconv"
	"strings"
)

type ExecParent struct {
	// "n" for ExecNode, "d" for Dataset
	Type string
	ID int

	// name of ExecNode output that is being input
	Name string

	// the data type of this parent
	DataType DataType
}

func (p ExecParent) String() string {
	var parts []string
	parts = append(parts, p.Type)
	parts = append(parts, strconv.Itoa(p.ID))
	if p.Type == "n" {
		parts = append(parts, p.Name)
	}
	return strings.Join(parts, ",")
}

type ExecInput struct {
	Name string
	// nil if input can be any type
	DataTypes []DataType
	// true if this node can accept multiple inputs for this name
	Variable bool
}

type ExecOutput struct {
	Name string
	DataType DataType
}

type ExecNode struct {
	ID int
	Name string
	Op string
	Params string

	// currently configured parents for each input
	Parents map[string][]ExecParent
}

func (node ExecNode) GetOp() ExecOpProvider {
	return GetExecOp(node.Op)
}

func (node ExecNode) GetInputs() []ExecInput {
	return node.GetOp().GetInputs(node.Params)
}

func (node ExecNode) GetInputTypes() map[string][]DataType {
	inputTypes := make(map[string][]DataType)
	for _, input := range node.GetInputs() {
		for _, parent := range node.Parents[input.Name] {
			inputTypes[input.Name] = append(inputTypes[input.Name], parent.DataType)
		}
	}
	return inputTypes
}

func (node ExecNode) GetOutputs() []ExecOutput {
	return node.GetOp().GetOutputs(node.Params, node.GetInputTypes())
}

func (node ExecNode) GetOutputTypes() map[string]DataType {
	outputTypes := make(map[string]DataType)
	for _, output := range node.GetOutputs() {
		outputTypes[output.Name] = output.DataType
	}
	return outputTypes
}

type Runnable struct {
	Name string
	Op string
	Params string
	InputDatasets map[string][]Dataset
	OutputDatasets map[string]Dataset
}

func (node Runnable) GetOp() ExecOpProvider {
	return GetExecOp(node.Op)
}
