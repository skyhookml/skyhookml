package skyhook

// A git repository that's used as a library in some component.
type PytorchRepository struct {
	URL string

	// Optional (empty string for latest commit in default branch)
	Commit string
}

type PytorchComponentParams struct {
	// code defining a pytorch nn.Module called "M"
	// forward pass takes some inputs, potentially some targets
	// and it returns a dict mapping names to tensors
	Code string

	// inputs/targets are provided as arguments to forward pass
	NumInputs int
	NumTargets int

	// produces these recommended skyhook outputs
	Outputs map[string]DataType

	// forward pass output dict also includes these layers and losses
	Layers []string
	Losses []string

	Repositories []PytorchRepository

	// TODO: some kind of preparation functions to support things like triplet loss
}

type PytorchComponent struct {
	ID int
	Name string
	Params PytorchComponentParams
}

type PytorchArchInput struct {
	// dataset or layer
	Type string

	ComponentIdx int
	Layer string

	DatasetIdx int
}

type PytorchArchParams struct {
	// datasets during training are numbered starting from inputs, then continuing with targets
	// DatasetIdx refer to this unified numbering scheme
	NumInputs int
	NumTargets int

	Components []struct{
		// PytorchComponent ID
		ID int
		// arbitrary JSON parameters
		Params string
		// where should component.Inputs come from
		// these must be layer or input dataset (not target dataset)
		Inputs []PytorchArchInput
		// where should component.Targets come from
		// these could be layer, input dataset, or target dataset
		Targets []PytorchArchInput
	}
	Losses []struct{
		ComponentIdx int
		Layer string
		Weight float64
	}
}

type PytorchArch struct {
	ID int
	Name string
	Params PytorchArchParams
}

type PytorchTrainParams struct {
	ArchID int

	// options for each input dataset
	// the number of inputs connected to this node should be NumInputs+NumTargets
	// (but we don't need options for all of them)
	InputOptions []struct{
		Idx int
		// JSON-encoded; structure depends on data type
		Value string
	}
}

type PytorchInferParams struct {
	ArchID int

	InputOptions []struct{
		Idx int
		Value string
	}

	OutputDatasets []struct{
		ComponentIdx int
		Layer string
		DataType DataType
	}
}
