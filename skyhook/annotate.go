package skyhook

type AnnotateDataset struct {
	ID int
	Dataset Dataset
	Inputs []ExecParent
	Tool string
	Params string
}
