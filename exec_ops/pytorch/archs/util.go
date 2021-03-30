package archs

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"github.com/skyhookml/skyhookml/exec_ops/pytorch"
)

type Impl struct {
	ID string
	Name string
	TrainInputs []skyhook.ExecInput
	InferInputs []skyhook.ExecInput
	InferOutputs []skyhook.ExecOutput
	TrainPrepare func(skyhook.Runnable) (skyhook.PytorchTrainParams, error)
	InferPrepare func(skyhook.Runnable) (skyhook.PytorchInferParams, error)
}

func AddImpl(impl Impl) {
	// We use pytorch.TrainImpl and pytorch.InferImpl as templates for creating
	// arch-specific exec op.
	trainImpl := pytorch.TrainImpl
	trainImpl.Config = skyhook.ExecOpConfig{
		ID: impl.ID+"_train",
		Name: impl.Name + " (train)",
		Description: impl.Name + " (train)",
	}
	trainImpl.Inputs = impl.TrainInputs
	trainImpl.Prepare = func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
		params, err := impl.TrainPrepare(node)
		if err != nil {
			return nil, err
		}
		node.Params = string(skyhook.JsonMarshal(params))

		// set input datasets: pytorch expects "inputs" and "models"
		inputDatasets := make(map[string][]skyhook.Dataset)
		for _, input := range impl.TrainInputs {
			for _, ds := range node.InputDatasets[input.Name] {
				if input.Name == "model" || input.Name == "models" {
					inputDatasets["models"] = append(inputDatasets["models"], ds)
				} else {
					inputDatasets["inputs"] = append(inputDatasets["inputs"], ds)
				}
			}
		}
		node.InputDatasets = inputDatasets

		node.Op = "pytorch_train"
		return pytorch.TrainImpl.Prepare(url, node)
	}

	inferImpl := pytorch.InferImpl
	inferImpl.Config = skyhook.ExecOpConfig{
		ID: impl.ID+"_infer",
		Name: impl.Name + " (infer)",
		Description: impl.Name + " (infer)",
	}
	inferImpl.Inputs = impl.InferInputs
	inferImpl.Outputs = impl.InferOutputs
	inferImpl.Prepare = func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
		params, err := impl.InferPrepare(node)
		if err != nil {
			return nil, err
		}
		node.Params = string(skyhook.JsonMarshal(params))

		// set input datasets: pytorch expects "inputs" and "model"
		inputDatasets := make(map[string][]skyhook.Dataset)
		for _, input := range impl.TrainInputs {
			for _, ds := range node.InputDatasets[input.Name] {
				if input.Name == "model" {
					inputDatasets["model"] = append(inputDatasets["model"], ds)
				} else {
					inputDatasets["inputs"] = append(inputDatasets["inputs"], ds)
				}
			}
		}
		node.InputDatasets = inputDatasets

		// set output datasets: should match GetInferOutputs(params)
		outputDatasets := make(map[string]skyhook.Dataset)
		expectedOutputs := pytorch.GetInferOutputs(params)
		for i, output := range inferImpl.Outputs {
			ds := node.OutputDatasets[output.Name]
			expectedName := expectedOutputs[i].Name
			outputDatasets[expectedName] = ds
		}
		node.OutputDatasets = outputDatasets

		node.Op = "pytorch_infer"
		return pytorch.InferImpl.Prepare(url, node)
	}

	skyhook.AddExecOpImpl(trainImpl)
	skyhook.AddExecOpImpl(inferImpl)
}
