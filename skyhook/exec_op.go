package skyhook

type ExecOp interface {
	Parallelism() int
	Apply(task ExecTask) error
	Close()
}

type ExecTask struct {
	// For incremental operations, this must be the output key that will be created by this task.
	// TODO: operation may need to produce multiple output keys at some task
	// For other operations, I think this can be arbitrary, but usually it's still related to the output key
	Key string
	Items []Item

	Metadata string
}

type ExecOpImpl struct {
	Requirements func(url string, node ExecNode) map[string]int
	GetTasks func(url string, node ExecNode, items [][]Item) ([]ExecTask, error)
	Prepare func(url string, node ExecNode, outputDatasets []Dataset) (ExecOp, error)

	// optional; if set, op is considered "incremental"
	Incremental bool
	GetOutputKeys func(node ExecNode, inputs [][]string) []string
	GetNeededInputs func(node ExecNode, outputs []string) [][]string

	// Docker image name
	ImageName func(url string, node ExecNode) (string, error)
}

var ExecOpImpls = make(map[string]ExecOpImpl)

func GetExecOpImpl(opName string) *ExecOpImpl {
	impl, ok := ExecOpImpls[opName]
	if !ok {
		return nil
	}
	return &impl
}
