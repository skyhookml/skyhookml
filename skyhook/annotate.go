package skyhook

type AnnotateDataset struct {
	ID int
	Dataset Dataset
	Inputs []Dataset
	Tool string
	Params string
}
