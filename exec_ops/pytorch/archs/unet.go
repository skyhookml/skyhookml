package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
)

func init() {
	type TrainParams struct {
		skyhook.PytorchTrainParams
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
		NumClasses int `json:"num_classes,omitempty"`
	}

	AddImpl(Impl{
		ID: "pytorch_unet",
		Name: "UNet",
		TrainInputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "labels", DataTypes: []skyhook.DataType{skyhook.ArrayType}},
			{Name: "models", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferInputs: []skyhook.ExecInput{
			{Name: "input", DataTypes: []skyhook.DataType{skyhook.ImageType, skyhook.VideoType}},
			{Name: "model", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferOutputs: []skyhook.ExecOutput{
			{Name: "output", DataType: skyhook.ArrayType},
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
				NumClasses: params.NumClasses,
			}
			p.Components = map[int]string{
				0: string(skyhook.JsonMarshal(modelParams)),
			}

			p.ArchID = "unet"
			return p, nil
		},
		InferPrepare: func(node skyhook.Runnable) (skyhook.PytorchInferParams, error) {
			var params InferParams
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return skyhook.PytorchInferParams{}, err
			}
			p := skyhook.PytorchInferParams{
				ArchID: "unet",
				OutputDatasets: []skyhook.PIOutputDataset{{
					ComponentIdx: 0,
					Layer: "classes",
					DataType: skyhook.ArrayType,
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
