package skyhook

import (
	"fmt"
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

// Config of the ExecOp for front-end.
type ExecOpConfig struct {
	ID string
	Name string
	Description string
}

type ExecOpProvider interface {
	// Returns config for front-end.
	Config() ExecOpConfig
	// Returns resource requirements.
	Requirements(node Runnable) map[string]int
	// Returns list of tasks.
	// items: is a map: input name -> input dataset index -> items in that dataset
	GetTasks(node Runnable, items map[string][][]Item) ([]ExecTask, error)
	// Prepare the ExecOp for a node.
	Prepare(url string, node Runnable) (ExecOp, error)
	// Determines the input specification of a node.
	GetInputs(params string) []ExecInput
	// Determines the output specification of a node.
	GetOutputs(params string, inputTypes map[string][]DataType) []ExecOutput

	// Incremental ops support partial computation of their outputs.
	// This is only possible for concrete nodes (Resolve must return nil).
	IsIncremental() bool
	// Must be implemented if Incremental.
	// GetOutputKeys returns all output keys that would be produced given a set of input keys.
	// GetNeededInputs returns the input keys that are needed to compute a given subset of output keys.
	GetOutputKeys(node ExecNode, inputs map[string][][]string) []string
	GetNeededInputs(node ExecNode, outputs []string) map[string][][]string

	// Docker image name
	GetImageName(node Runnable) (string, error)

	// Optional system to provide customized state to store in ExecNode jobs.
	// For example, when training a model, we may want to store the loss history.
	// Can return nil to use defaults.
	// Second return value is the view of the JobOp, empty string to use default view.
	GetJobOp(node Runnable) (JobOp, string)

	// Virtualize is called when constructing an initial ExecutionGraph.
	// For example, if(A) { input B } else { input C } can be implemented by:
	// - Virtualize should return VirtualNode requiring only A
	// - Resolve can load A, and output a new graph that includes B or C depending on A
	Virtualize(node ExecNode) *VirtualNode

	// Optional system for pre-processing steps, dynamic execution graphs, etc.
	// Given a VirtualNode, returns a subgraph of new VirtualNodes that implement it.
	// Or nil if the VirtualNode is already OK.
	// Resolve is called just before executing the node.
	Resolve(node *VirtualNode, inputDatasets map[string][]Dataset, items map[string][][]Item) ExecutionGraph
}

// A helper to implement an ExecOpProvider as a struct.
// This way optional methods can be omitted and defaults used instead.
// It can be compiled to an ExecOpProvider by wrapping in an ExecOpImplProvider.
type ExecOpImpl struct {
	Config ExecOpConfig
	Requirements func(node Runnable) map[string]int
	GetTasks func(node Runnable, items map[string][][]Item) ([]ExecTask, error)
	Prepare func(url string, node Runnable) (ExecOp, error)

	// only one should be set (static/dynamic)
	ImageName string
	GetImageName func(node Runnable) (string, error)

	// static specification of inputs/outputs
	// one of dynamic/static should be set
	Inputs []ExecInput
	Outputs []ExecOutput

	// dynamic specification of inputs/outputs
	// one of dynamic/static should be set
	GetInputs func(params string) []ExecInput
	GetOutputs func(params string, inputTypes map[string][]DataType) []ExecOutput

	// optional; if set, op is considered "incremental"
	Incremental bool
	GetOutputKeys func(node ExecNode, inputs map[string][][]string) []string
	GetNeededInputs func(node ExecNode, outputs []string) map[string][][]string

	// more various optional functions
	GetJobOp func(node Runnable) (JobOp, string)
	Virtualize func(node ExecNode) *VirtualNode
	Resolve func(node *VirtualNode, inputDatasets map[string][]Dataset, items map[string][][]Item) ExecutionGraph
}

type ExecOpImplProvider struct {
	Impl ExecOpImpl
}
func (p ExecOpImplProvider) Config() ExecOpConfig {
	return p.Impl.Config
}
func (p ExecOpImplProvider) Requirements(node Runnable) map[string]int {
	return p.Impl.Requirements(node)
}
func (p ExecOpImplProvider) GetTasks(node Runnable, items map[string][][]Item) ([]ExecTask, error) {
	return p.Impl.GetTasks(node, items)
}
func (p ExecOpImplProvider) Prepare(url string, node Runnable) (ExecOp, error) {
	return p.Impl.Prepare(url, node)
}
func (p ExecOpImplProvider) GetInputs(params string) []ExecInput {
	if p.Impl.Inputs != nil {
		return p.Impl.Inputs
	} else {
		return p.Impl.GetInputs(params)
	}
}
func (p ExecOpImplProvider) GetOutputs(params string, inputTypes map[string][]DataType) []ExecOutput {
	if p.Impl.Outputs != nil {
		return p.Impl.Outputs
	} else {
		return p.Impl.GetOutputs(params, inputTypes)
	}
}
func (p ExecOpImplProvider) IsIncremental() bool {
	return p.Impl.Incremental
}
func (p ExecOpImplProvider) GetOutputKeys(node ExecNode, inputs map[string][][]string) []string {
	return p.Impl.GetOutputKeys(node, inputs)
}
func (p ExecOpImplProvider) GetNeededInputs(node ExecNode, outputs []string) map[string][][]string {
	return p.Impl.GetNeededInputs(node, outputs)
}
func (p ExecOpImplProvider) GetImageName(node Runnable) (string, error) {
	if p.Impl.ImageName != "" {
		return p.Impl.ImageName, nil
	} else {
		return p.Impl.GetImageName(node)
	}
}
func (p ExecOpImplProvider) GetJobOp(node Runnable) (JobOp, string) {
	if p.Impl.GetJobOp == nil {
		return nil, ""
	}
	return p.Impl.GetJobOp(node)
}
func (p ExecOpImplProvider) Resolve(node *VirtualNode, inputDatasets map[string][]Dataset, items map[string][][]Item) ExecutionGraph {
	if p.Impl.Resolve == nil {
		return nil
	}
	return p.Impl.Resolve(node, inputDatasets, items)
}

func (p ExecOpImplProvider) Virtualize(node ExecNode) *VirtualNode {
	if p.Impl.Virtualize != nil {
		return p.Impl.Virtualize(node)
	}
	parents := make(map[string][]VirtualParent)
	for name := range node.Parents {
		parents[name] = make([]VirtualParent, len(node.Parents[name]))
		for i := range parents[name] {
			execParent := node.Parents[name][i]
			var graphID GraphID
			if execParent.Type == "n" {
				graphID.Type = "exec"
			} else if execParent.Type == "d" {
				graphID.Type = "dataset"
			}
			graphID.ID = execParent.ID
			parents[name][i] = VirtualParent{
				GraphID: graphID,
				Name: execParent.Name,
				DataType: execParent.DataType,
			}
		}
	}
	return &VirtualNode{
		Name: node.Name,
		Op: node.Op,
		Params: node.Params,
		Parents: parents,
		OrigNode: node,
	}
}

var ExecOpProviders = make(map[string]ExecOpProvider)

func AddExecOpImpl(impl ExecOpImpl) {
	id := impl.Config.ID
	if ExecOpProviders[id] != nil {
		panic(fmt.Errorf("conflicting provider %s", id))
	}
	ExecOpProviders[id] = ExecOpImplProvider{impl}
}

func GetExecOp(opName string) ExecOpProvider {
	provider := ExecOpProviders[opName]
	if provider == nil {
		panic(fmt.Errorf("no such provider %s", opName))
	}
	return provider
}
