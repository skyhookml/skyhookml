package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
)

func init() {
	type TrainParams struct {
		skyhook.PytorchTrainParams
		Mode string
		Resize skyhook.PDDImageOptions
		ValPercent int
	}

	type InferParams struct {
		Resize skyhook.PDDImageOptions
		ConfidenceThreshold float64
	}

	type ModelParams struct {
		Mode string `json:"mode,omitempty"`
		ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`
		IouThreshold float64 `json:"iou_threshold,omitempty"`
	}

	AddImpl(Impl{
		ID: "pytorch_yolov5",
		Name: "YOLOv5-pytorch",
		TrainInputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "detections", DataTypes: []skyhook.DataType{skyhook.DetectionType}},
			{Name: "models", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferInputs: []skyhook.ExecInput{
			{Name: "input", DataTypes: []skyhook.DataType{skyhook.ImageType, skyhook.VideoType}},
			{Name: "model", DataTypes: []skyhook.DataType{skyhook.FileType}},
		},
		InferOutputs: []skyhook.ExecOutput{
			{Name: "detections", DataType: skyhook.DetectionType},
		},
		TrainPrepare: func(node skyhook.Runnable) (skyhook.PytorchTrainParams, error) {
			var params TrainParams
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return skyhook.PytorchTrainParams{}, err
			}
			p := params.PytorchTrainParams
			p.Dataset.Op = "default"
			p.Dataset.Params = string(skyhook.JsonMarshal(skyhook.PDDParams{
				InputOptions: []interface{}{params.Resize, struct{}{}},
				ValPercent: params.ValPercent,
			}))

			modelParams := ModelParams{
				Mode: params.Mode,
			}
			p.Components = map[int]string{
				0: string(skyhook.JsonMarshal(modelParams)),
			}

			p.ArchID = "yolov5"
			return p, nil
		},
		InferPrepare: func(node skyhook.Runnable) (skyhook.PytorchInferParams, error) {
			var params InferParams
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return skyhook.PytorchInferParams{}, err
			}
			p := skyhook.PytorchInferParams{
				ArchID: "yolov5",
				OutputDatasets: []skyhook.PIOutputDataset{{
					ComponentIdx: 0,
					Layer: "detections",
					DataType: skyhook.DetectionType,
				}},
				InputOptions: []skyhook.PIInputOption{{
					Idx: 0,
					Value: string(skyhook.JsonMarshal(params.Resize)),
				}},
			}

			modelParams := ModelParams{
				ConfidenceThreshold: params.ConfidenceThreshold,
			}
			p.Components = map[int]string{
				0: string(skyhook.JsonMarshal(modelParams)),
			}

			return p, nil
		},
	})
}
