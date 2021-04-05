package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"encoding/json"
	"fmt"
)

func init() {
	type TrainParams struct {
		skyhook.PytorchTrainParams
		Mode string
		Width int
		Height int
		ValPercent int
	}

	type InferParams struct {
		Width int
		Height int
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
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				return skyhook.PytorchTrainParams{}, fmt.Errorf("node is not configured: %v", err)
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
			}
			p.Components = map[int]string{
				0: string(skyhook.JsonMarshal(modelParams)),
			}

			p.ArchID = "yolov5"
			return p, nil
		},
		InferPrepare: func(node skyhook.Runnable) (skyhook.PytorchInferParams, error) {
			var params InferParams
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				return skyhook.PytorchInferParams{}, fmt.Errorf("node is not configured: %v", err)
			}
			p := skyhook.PytorchInferParams{
				ArchID: "yolov5",
				OutputDatasets: []skyhook.PIOutputDataset{{
					ComponentIdx: 0,
					Layer: "detections",
					DataType: skyhook.DetectionType,
				}},
			}
			if params.Width > 0 || params.Height > 0 {
				opt := skyhook.PDDImageOptions{params.Width, params.Height}
				p.InputOptions = []skyhook.PIInputOption{{
					Idx: 0,
					Value: string(skyhook.JsonMarshal(opt)),
				}}
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
