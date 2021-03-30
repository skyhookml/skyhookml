package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"encoding/json"
	"fmt"
)

func init() {
	type TrainParams struct {
		skyhook.PytorchTrainParams
		Width int
		Height int
		ValPercent int
	}

	AddImpl(Impl{
		ID: "pytorch_yolov3",
		Name: "YOLOv3-pytorch",
		TrainInputs: []skyhook.ExecInput{
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.ImageType}},
			{Name: "detections", DataTypes: []skyhook.DataType{skyhook.DetectionType}},
			{Name: "models", DataTypes: []skyhook.DataType{skyhook.StringType}},
		},
		InferInputs: []skyhook.ExecInput{
			{Name: "input", DataTypes: []skyhook.DataType{skyhook.ImageType, skyhook.VideoType}},
			{Name: "model", DataTypes: []skyhook.DataType{skyhook.StringType}},
		},
		InferOutputs: []skyhook.ExecOutput{
			{Name: "detections", DataType: skyhook.DetectionType},
		},
		TrainPrepare: func(node skyhook.Runnable) (skyhook.PytorchTrainParams, error) {
			var params TrainParams
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				return skyhook.PytorchTrainParams{}, fmt.Errorf("node is not configured")
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
			p.Arch = "yolov3"
			return p, nil
		},
		InferPrepare: func(node skyhook.Runnable) (skyhook.PytorchInferParams, error) {
			var params skyhook.PytorchInferParams
			if err := json.Unmarshal([]byte(node.Params), &params); err != nil {
				return skyhook.PytorchInferParams{}, fmt.Errorf("node is not configured")
			}
			return params, nil
		},
	})
}
