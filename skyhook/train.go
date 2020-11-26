package skyhook

type KerasArchParams struct {
	Inputs [][2]string
	Arch [][2]string
	Outputs []string
}

type KerasArch struct {
	ID int
	Name string
	Params KerasArchParams
}

type TrainNode struct {
	ID int
	Name string
	Op string
	Params string
	ParentIDs []int
	Outputs []DataType
	Trained bool
}

// Params for a exec node that infers using model.
type ModelExecParams struct {
	TrainNodeID int
}
