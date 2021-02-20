package skyhook

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

type PytorchNodeParams struct {
	ArchID int

	// IDs of other TrainNodes to load model from
	// TODO: should probably specify some kind of names mapping
	// (if arch has only one instance of each component, then mapping should
	// be clear, but need mapping in case arch has multiple of the same component)
	LoadFrom []int

	// input datasets, length equals NumInputs+NumTargets
	InputDatasets []struct{
		ID int

		// JSON options, structure depends on data type
		Options string
	}

	OutputDatasets []struct{
		ComponentIdx int
		Layer string
		DataType DataType
	}
}
