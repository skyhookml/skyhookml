package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
)

func init() {
	type TrainParams struct {
		skyhook.PytorchTrainParams
		Mode string
		Width int
		Height int
		NumClasses int
		ValPercent int
	}

	type InferParams struct {
		Width int
		Height int
	}

	type ModelParams struct {
		// "resnet15", "resnet34", etc. (see python/skyhook/pytorch/components/resnet.py)
		Mode string `json:"mode,omitempty"`

		NumClasses int `json:"num_classes,omitempty"`
	}

	AddImpl(Impl{
		ID: "pytorch_resnet",
		Name: "ResNet",
		TrainInputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "labels", DataTypes: []skyhook.DataType{skyhook.IntType}},
			{Name: "models", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferInputs: []skyhook.ExecInput{
			{Name: "input", DataTypes: []skyhook.DataType{skyhook.ImageType, skyhook.VideoType}},
			{Name: "model", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferOutputs: []skyhook.ExecOutput{
			{Name: "categories", DataType: skyhook.IntType},
		},
		TrainPrepare: func(node skyhook.Runnable) (skyhook.PytorchTrainParams, error) {
			var params TrainParams
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return skyhook.PytorchTrainParams{}, err
			}
			p := params.PytorchTrainParams
			p.Dataset.Op = "default"
			p.Dataset.Params = string(skyhook.JsonMarshal(skyhook.PDDParams{
				InputOptions: []interface{}{skyhook.PDDImageOptions{
					Width: params.Width,
					Height: params.Height,
				}, struct{}{}},
				ValPercent: params.ValPercent,
			}))

			modelParams := ModelParams{
				Mode: params.Mode,
				NumClasses: params.NumClasses,
			}
			p.Components = map[int]string{
				0: string(skyhook.JsonMarshal(modelParams)),
			}

			p.ArchID = "resnet"
			return p, nil
		},
		InferPrepare: func(node skyhook.Runnable) (skyhook.PytorchInferParams, error) {
			var params InferParams
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return skyhook.PytorchInferParams{}, err
			}
			p := skyhook.PytorchInferParams{
				ArchID: "resnet",
				OutputDatasets: []skyhook.PIOutputDataset{{
					ComponentIdx: 0,
					Layer: "cls",
					DataType: skyhook.IntType,
				}},
			}
			if params.Width > 0 || params.Height > 0 {
				opt := skyhook.PDDImageOptions{params.Width, params.Height}
				p.InputOptions = []skyhook.PIInputOption{{
					Idx: 0,
					Value: string(skyhook.JsonMarshal(opt)),
				}}
			}
			return p, nil
		},
	})
}
