package skyhook

import (
	"crypto/sha256"
	"encoding/hex"
)

// A git repository that's used as a library in some component.
type PytorchRepository struct {
	URL string

	// Optional (empty string for latest commit in default branch)
	Commit string
}

func (repo PytorchRepository) Hash() string {
	// compute hash as sha256(url[@commit])
	h := sha256.New()
	h.Write([]byte(repo.URL))
	if repo.Commit != "" {
		h.Write([]byte("@"+repo.Commit))
	}
	bytes := h.Sum(nil)
	hash := hex.EncodeToString(bytes)
	return hash
}

type PytorchComponentParams struct {
	// the module can be defined one of three ways:
	// - a built-in module in exec_ops/pytorch/models/X.py
	// - a module X in a git repository Y
	// - hardcoded
	// only one of BuiltInModule, RepositoryModule, and Code should be set
	// if RepositoryModule is set, Repository must be as well
	Module struct {
		BuiltInModule string

		Repository PytorchRepository
		RepositoryModule string

		Code string
	}

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
	ID string
	Params PytorchComponentParams
}

type PytorchArchInput struct {
	// dataset or layer
	Type string

	ComponentIdx int
	Layer string

	DatasetIdx int
}

type PytorchArchComponent struct {
	// PytorchComponent ID
	ID string
	// arbitrary JSON parameters
	Params string
	// where should component.Inputs come from
	// these must be layer or input dataset (not target dataset)
	Inputs []PytorchArchInput
	// where should component.Targets come from
	// these could be layer, input dataset, or target dataset
	Targets []PytorchArchInput
}

type PytorchArchLoss struct {
	ComponentIdx int
	Layer string
	Weight float64
}

type PytorchArchParams struct {
	// datasets during training are numbered starting from inputs, then continuing with targets
	// DatasetIdx refer to this unified numbering scheme
	NumInputs int
	NumTargets int

	Components []PytorchArchComponent
	Losses []PytorchArchLoss
	Scores []PytorchArchLoss
}

type PytorchArch struct {
	ID string
	Params PytorchArchParams
}

// Pytorch default dataset parameters.
// This configures skyhook.pytorch.dataset.default,
// and goes JSON-encoded in PytorchTrainParams.Dataset.Params.
type PDDImageOptions struct {
	Width int
	Height int
}
type PDDParams struct {
	InputOptions []interface{}
	ValPercent int
}

// Pytorch restore models
type PRestore struct {
	SrcPrefix string
	DstPrefix string
	SkipPrefixes string
}

type PytorchOpParams struct {
	Op string
	Params string
}

type PytorchTrainParams struct {
	ArchID string
	// dataset options
	Dataset PytorchOpParams
	// data augmentation
	Augment []PytorchOpParams
	// restoring models, corresponding to the "models" input
	Restore []PRestore
	// training loop parameters
	Train PytorchOpParams
	// Configuring component parameters.
	// This is like PytorchArchComponent.Params, but just overrides or extends
	// those parameters with more options.
	// It is map from component index to JSON-encoded parameters.
	Components map[int]string
}

type PIInputOption struct {
	Idx int
	Value string
}
type PIOutputDataset struct {
	ComponentIdx int
	Layer string
	DataType DataType
}

type PytorchInferParams struct {
	ArchID string
	InputOptions []PIInputOption
	OutputDatasets []PIOutputDataset
	// See PytorchTrainParams.Components
	Components map[int]string
}
