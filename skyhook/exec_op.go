package skyhook

import (
	"runtime"
)

type ExecOp interface {
	Parallelism() int
	Apply(task ExecTask) error
	Close()
}

// A wrapper for a simple exec op that needs no persistent state.
// So the wrapper just wraps a function, along with desired parallelism.
type SimpleExecOp struct {
	ApplyFunc func(ExecTask) error
	P int
}
func (e SimpleExecOp) Parallelism() int {
	if e.P == 0 {
		return runtime.NumCPU()
	}
	return e.P
}
func (e SimpleExecOp) Apply(task ExecTask) error {
	return e.ApplyFunc(task)
}
func (e SimpleExecOp) Close() {}

type ExecTask struct {
	// For incremental operations, this must be the output key that will be created by this task.
	// TODO: operation may need to produce multiple output keys at some task
	// For other operations, I think this can be arbitrary, but usually it's still related to the output key
	Key string

	// Generally maps from input name to list of items in each dataset at that input
	Items map[string][][]Item

	Metadata string
}

type ExecOpImpl struct {
	Requirements func(node Runnable) map[string]int
	// items is: input name -> input dataset index -> items in that dataset
	GetTasks func(node Runnable, items map[string][][]Item) ([]ExecTask, error)
	// initialize an ExecOp
	Prepare func(url string, node Runnable) (ExecOp, error)
	// determine the output names/types given current inputs and configuration
	GetOutputs func(params string, inputTypes map[string][]DataType) []ExecOutput

	// optional; if set, op is considered "incremental"
	Incremental bool
	GetOutputKeys func(node ExecNode, inputs map[string][][]string) []string
	GetNeededInputs func(node ExecNode, outputs []string) map[string][][]string

	// Docker image name
	ImageName func(node Runnable) (string, error)

	// Optional system to provide customized state to store in ExecNode jobs.
	// For example, when training a model, we may want to store the loss history.
	GetJobOp func(node Runnable) JobOp

	// Optional system for dynamic control flow.
	// Virtualize is called when constructing an initial ExecutionGraph.
	// For example, if(A) { input B } else { input C } can be implemented by:
	// - Virtualize should return VirtualNode requiring only A
	// - Resolve can load A, and output a new graph that includes B or C depending on A
	Virtualize func(node ExecNode) *VirtualNode

	// Optional system for pre-processing steps, dynamic execution graphs, etc.
	// Given a VirtualNode, returns a subgraph of new VirtualNodes that implement it.
	// Or nil if the VirtualNode is already OK.
	// Resolve is called just before executing the node.
	Resolve func(node *VirtualNode, inputDatasets map[string][]Dataset, items map[string][][]Item) ExecutionGraph
}

func Virtualize(node ExecNode) *VirtualNode {
	opImpl := GetExecOpImpl(node.Op)
	if opImpl.Virtualize != nil {
		return opImpl.Virtualize(node)
	}
	parents := make([][]VirtualParent, len(node.Parents))
	for i := range parents {
		parents[i] = make([]VirtualParent, len(node.Parents[i]))
		for j := range parents[i] {
			execParent := node.Parents[i][j]
			var graphID GraphID
			if execParent.Type == "n" {
				graphID.Type = "exec"
			} else if execParent.Type == "d" {
				graphID.Type = "dataset"
			}
			graphID.ID = execParent.ID
			parents[i][j] = VirtualParent{
				GraphID: graphID,
				Name: execParent.Name,
			}
		}
	}
	return &VirtualNode{
		Name: node.Name,
		Op: node.Op,
		Params: node.Params,
		Inputs: node.Inputs,
		Outputs: node.Outputs,

		Parents: parents,
		OrigNode: node,
	}
}

var ExecOpImpls = make(map[string]ExecOpImpl)

func GetExecOpImpl(opName string) *ExecOpImpl {
	impl, ok := ExecOpImpls[opName]
	if !ok {
		return nil
	}
	return &impl
}
