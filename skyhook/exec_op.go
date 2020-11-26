package skyhook

type ExecOp interface {
	Apply(key string, items []Item) (map[string][]Data, error)
	Close()
}

type ExecOpImpl struct {
	Requirements func(url string, node ExecNode) map[string]int
	Prepare func(url string, node ExecNode) (ExecOp, error)
}

var ExecOpImpls = make(map[string]ExecOpImpl)

func GetExecOpImpl(opName string) *ExecOpImpl {
	impl, ok := ExecOpImpls[opName]
	if !ok {
		return nil
	}
	return &impl
}
