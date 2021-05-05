package pytorch

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/exec_ops/python"

	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type TrainOp struct {
	url string
	node skyhook.Runnable
	outputDataset skyhook.Dataset
}

func (e *TrainOp) Parallelism() int {
	return 1
}

func (e *TrainOp) Apply(task skyhook.ExecTask) error {
	// Prepare parameters needed by Python script.
	var params skyhook.PytorchTrainParams
	if err := exec_ops.DecodeParams(e.node, &params, false); err != nil {
		return err
	}
	arch, components, err := GetTrainArgs(e.url, params.ArchID)
	if err != nil {
		return err
	}

	// Fetch any git repositories that aren't already available locally.
	if err := EnsureRepositories(components); err != nil {
		return err
	}

	inputDatasets := e.node.InputDatasets
	e.outputDataset.Mkdir()

	// Initialize HTTP server that will serve the training parameters.
	muxFunc := func(mux *http.ServeMux) error {
		var trainConfig struct {
			Params skyhook.PytorchTrainParams
			Arch *skyhook.PytorchArch
			Components map[string]*skyhook.PytorchComponent
			Inputs []skyhook.Dataset
			ParentModels []skyhook.Dataset
			Output skyhook.Dataset
			TrainSplit []string
			ValidSplit []string
		}
		trainConfig.Params = params
		trainConfig.Arch = arch
		trainConfig.Components = components
		trainConfig.Inputs = inputDatasets["inputs"]
		trainConfig.ParentModels = inputDatasets["models"]
		trainConfig.Output = e.outputDataset
		if len(inputDatasets["train_split"]) > 0 || len(inputDatasets["valid_split"]) > 0 {
			// If either/both train/valid split are set, then we need to provide keys
			// for train/valid.
			// (1) Get all keys across the input datasets.
			// (2) Get specified train and/or valid keys.
			// (3) Compute missing split if needed. (If only one of train/valid is given.)
			allItems, err := exec_ops.GetItems(e.url, inputDatasets["inputs"])
			if err != nil {
				return err
			}

			trainKeys := make(map[string]bool)
			validKeys := make(map[string]bool)
			if len(inputDatasets["train_split"]) > 0 {
				items, err := exec_ops.GetDatasetItems(e.url, inputDatasets["train_split"][0])
				if err != nil {
					return err
				}
				for key := range items {
					if allItems[key] == nil {
						continue
					}
					trainKeys[key] = true
				}
			}
			if len(inputDatasets["valid_split"]) > 0 {
				items, err := exec_ops.GetDatasetItems(e.url, inputDatasets["valid_split"][0])
				if err != nil {
					return err
				}
				for key := range items {
					if allItems[key] == nil {
						continue
					}
					validKeys[key] = true
				}
			}

			if len(trainKeys) == 0 && len(validKeys) == 0 {
				return fmt.Errorf("train and/or valid split provided but dataset is empty")
			}

			if len(trainKeys) == 0 {
				for key := range allItems {
					if !validKeys[key] {
						trainKeys[key] = true
					}
				}
			} else if len(validKeys) == 0 {
				for key := range allItems {
					if !trainKeys[key] {
						validKeys[key] = true
					}
				}
			}

			for key := range trainKeys {
				trainConfig.TrainSplit = append(trainConfig.TrainSplit, key)
			}
			for key := range validKeys {
				trainConfig.ValidSplit = append(trainConfig.ValidSplit, key)
			}
		}

		mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
			skyhook.JsonResponse(w, trainConfig)
		})

		return nil
	}
	httpServer, err := python.NewHttpServer(e.url, muxFunc)
	if err != nil {
		return err
	}
	defer httpServer.Close()

	// Determine if automatic batch size reduction is enabled.
	// Also extract the desired batch size.
	autoBatchSize := false
	batchSize := 1
	if params.Train.Op == "default" {
		var trainParams skyhook.PTDParams
		if err := json.Unmarshal([]byte(params.Train.Params), &trainParams); err != nil {
			return err
		}
		autoBatchSize = trainParams.AutoBatchSize
		batchSize = trainParams.BatchSize
	}

	// Helper function to run train.py.
	// We use a function here since we may run train.py multiple times if we run
	// out of memory and AutoBatchSize is enabled.
	runTrain := func(batchSize int) error {
		cmd := exec.Command(
			"python3", "exec_ops/pytorch/train.py",
			e.url, strconv.Itoa(httpServer.Port), strconv.Itoa(batchSize),
		)
		cmd.Stdout = os.Stdout

		// Need to capture stderr output to determine if it includes "out of memory".
		// We use a TeeReader so that we can capture output while still redirecting it to os.Stderr.
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		stderrTee := io.TeeReader(stderr, os.Stderr)

		if err := cmd.Start(); err != nil {
			return err
		}

		// Launch goroutine to read from stderrTee and return the captured lines via channel.
		stderrCh := make(chan []string)
		go func() {
			var lines []string
			scanner := bufio.NewScanner(stderrTee)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			stderrCh <- lines
		}()

		err = cmd.Wait()
		stderrLines := <- stderrCh
		if err != nil {
			// Training failed.
			// But report this error as "out of memory" if that string appeared in the standard error output.
			for _, line := range stderrLines {
				if strings.Contains(line, "out of memory") {
					return fmt.Errorf("out of memory")
				}
			}
			return err
		}
		return nil
	}

	for {
		err := runTrain(batchSize)
		if err != nil {
			if autoBatchSize && strings.Contains(err.Error(), "out of memory") && batchSize > 1 {
				log.Printf("Training failed due to out of memory error, but automatic batch size reduction is enabled")
				log.Printf("Reducing batch size from %d to %d and trying again", batchSize, batchSize/2)
				batchSize /= 2
				continue
			} else {
				return err
			}
		}
		break
	}

	// add to the file dataset
	fileMetadata := skyhook.FileMetadata{Filename: "model.pt"}
	_, err = exec_ops.AddItem(e.url, e.outputDataset, "model", "pt", "", fileMetadata)
	if err != nil {
		return err
	}

	return nil
}

func (e *TrainOp) Close() {}

// Save losses from "jsonloss" lines in the pytorch train output.
type TrainJobOp struct {
	state skyhook.ModelJobState
}
const LossSignature string = "jsonloss"
func (op *TrainJobOp) Update(lines []string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, LossSignature) {
			continue
		}
		line = line[len(LossSignature):]
		// map from train/val -> loss name -> loss value
		var data map[string]map[string]float64
		skyhook.JsonUnmarshal([]byte(line), &data)
		op.state.TrainLoss = append(op.state.TrainLoss, data["train"]["loss"])
		op.state.ValLoss = append(op.state.ValLoss, data["val"]["loss"])
	}
}
func (op *TrainJobOp) Encode() string {
	return string(skyhook.JsonMarshal(op.state))
}
func (op *TrainJobOp) Stop() error {
	// handled by ExecJobOp
	return nil
}

var TrainImpl = skyhook.ExecOpImpl{
	Config: skyhook.ExecOpConfig{
		ID: "pytorch_train",
		Name: "Pytorch (train)",
		Description: "Pytorch (train)",
	},
	Inputs: []skyhook.ExecInput{
		{Name: "inputs", Variable: true},
		{Name: "models", DataTypes: []skyhook.DataType{skyhook.FileType}, Variable: true},
		{Name: "train_split"},
		{Name: "valid_split"},
	},
	Outputs: []skyhook.ExecOutput{{Name: "model", DataType: skyhook.FileType}},
	Requirements: func(node skyhook.Runnable) map[string]int {
		return nil
	},
	GetTasks: exec_ops.SingleTask("model"),
	Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
		op := &TrainOp{
			url: url,
			node: node,
			outputDataset: node.OutputDatasets["model"],
		}
		return op, nil
	},
	ImageName: "skyhookml/pytorch",
	GetJobOp: func(node skyhook.Runnable) (skyhook.JobOp, string) {
		return &TrainJobOp{}, "pytorch_train"
	},
	Resolve: func(node *skyhook.VirtualNode, inputDatasets map[string][]skyhook.Dataset, items map[string][][]skyhook.Item) skyhook.ExecutionGraph {
		// If parent items include non-materialized data (non-default provider),
		// then we need to run materialize op on those datasets.

		// list of names and indices that need materialization
		type ParentSpec struct {
			Name string
			Index int
		}
		var needed []ParentSpec
		for name, itemLists := range items {
			for idx, itemList := range itemLists {
				ok := true
				for _, item := range itemList {
					if item.Provider != nil {
						ok = false
						break
					}
				}
				if ok {
					continue
				}
				needed = append(needed, ParentSpec{
					Name: name,
					Index: idx,
				})
			}
		}

		if len(needed) == 0 {
			return nil
		}

		subgraph := make(skyhook.ExecutionGraph)
		origGID := node.GraphID()

		// create a materialize node to materialize the needed ones
		var matParents []skyhook.VirtualParent
		var matInputTypes []skyhook.DataType
		specToMatOutputIndex := make(map[ParentSpec]int)
		for i, spec := range needed {
			matParents = append(matParents, node.Parents[spec.Name][spec.Index])
			matInputTypes = append(matInputTypes, inputDatasets[spec.Name][spec.Index].DataType)
			specToMatOutputIndex[spec] = i
		}
		matGID := skyhook.GraphID{
			Type: origGID.Type,
			ID: origGID.ID,
			VirtualKey: origGID.VirtualKey+"/materialize",
		}
		subgraph[matGID] = &skyhook.VirtualNode{
			Name: node.Name+"-materialize",
			Op: "materialize",
			Params: "",
			Parents: map[string][]skyhook.VirtualParent{"inputs": matParents},
			OrigNode: node.OrigNode,
			VirtualKey: matGID.VirtualKey,
		}

		// and we need to update the pytorch node to input from the materialize node
		for name := range node.Parents {
			for idx := range node.Parents[name] {
				matOutputIndex, ok := specToMatOutputIndex[ParentSpec{name, idx}]
				if !ok {
					continue
				}
				node.Parents[name][idx] = skyhook.VirtualParent{
					GraphID: matGID,
					Name: fmt.Sprintf("outputs%d", matOutputIndex),
				}
			}
		}
		subgraph[origGID] = node

		return subgraph
	},
}

func init() {
	skyhook.AddExecOpImpl(TrainImpl)
}
