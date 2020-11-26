package skyhook

type TrainOp struct {
	Requirements func(url string, node TrainNode) map[string]int
	Train func(url string, node TrainNode) error
	Prepare func(url string, trainNode TrainNode, execNode ExecNode) (ExecOp, error)
}

var TrainOps = make(map[string]TrainOp)

func GetTrainOp(opName string) *TrainOp {
	op, ok := TrainOps[opName]
	if !ok {
		return nil
	}
	return &op
}
