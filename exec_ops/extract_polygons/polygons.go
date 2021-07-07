package extract_polygons

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/exec_ops/python"
)

type Params struct {
	DenoiseSize int `json:",omitempty"`
	GrowSize int `json:",omitempty"`
	SimplifyThreshold int `json:",omitempty"`
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "extract_polygons",
			Name: "Extract Polygons",
			Description: "Extract polygon shapes (Shape) from the output of a segmentation model (Array)",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.ArrayType}}},
		Outputs: []skyhook.ExecOutput{{Name: "shapes", DataType: skyhook.ShapeType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			return exec_ops.SimpleTasks(node, map[string][][]skyhook.Item{"inputs": rawItems["input"]})
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			pyParams := python.Params{
				Code: node.Params,
				Outputs: []skyhook.ExecOutput{{Name: "shapes", DataType: skyhook.ShapeType}},
			}

			cmd := skyhook.Command(
				"extract-polygons-"+node.Name,
				skyhook.CommandOptions{AllStderrLines: true},
				"python3", "exec_ops/extract_polygons/run.py",
			)
			return python.NewPythonOp(cmd, url, pyParams, node.InputDatasets["input"], []skyhook.Dataset{node.OutputDatasets["shapes"]})
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: "skyhookml/basic",
	})
}
