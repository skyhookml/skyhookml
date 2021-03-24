package reid

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"github.com/skyhookml/skyhookml/exec_ops/pytorch"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type TrainOp struct {
	url string
	node skyhook.Runnable
	dataset skyhook.Dataset
}

func (e *TrainOp) Parallelism() int {
	return 1
}

func (e *TrainOp) Apply(task skyhook.ExecTask) error {
	var params skyhook.PytorchTrainParams
	skyhook.JsonUnmarshal([]byte(e.node.Params), &params)
	arch, components, err := pytorch.GetTrainArgs(e.url, params.ArchID)
	if err != nil {
		return err
	}

	if err := pytorch.EnsureRepositories(components); err != nil {
		return err
	}

	videoDataset := e.node.InputDatasets["video"][0]
	detectionDataset := e.node.InputDatasets["detections"][0]
	datasetList := []skyhook.Dataset{videoDataset, detectionDataset}

	// pre-process the detections
	items, err := exec_ops.GetItems(e.url, datasetList)
	if err != nil {
		return err
	}
	matchesPath := filepath.Join(os.TempDir(), fmt.Sprintf("reid-%d", e.dataset.ID))
	if err := os.Mkdir(matchesPath, 0755); err != nil {
		return fmt.Errorf("could not mkdir %s: %v", matchesPath, err)
	}
	defer func() {
		//os.RemoveAll(matchesPath)
	}()
	for _, l := range items {
		key := l[1].Key
		log.Printf("[reid] pre-processing key %s", key)
		labelData, err := l[1].LoadData()
		if err != nil {
			return fmt.Errorf("error loading label (detection) data: %v", err)
		}
		detections := labelData.(skyhook.DetectionData).Detections
		matches := PreprocessMatches(detections)
		var matchList [][4]int
		for k, v := range matches {
			for _, id := range v {
				matchList = append(matchList, [4]int{k[0], k[1], k[2], id})
			}
		}
		bytes := skyhook.JsonMarshal(matchList)
		matchFname := filepath.Join(matchesPath, fmt.Sprintf("%s.json", key))
		if err := ioutil.WriteFile(matchFname, bytes, 0644); err != nil {
			return fmt.Errorf("error writing match data: %v", err)
		}
	}

	paramsArg := e.node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasetList))
	fmt.Println(e.dataset.ID, e.url, paramsArg, archArg, compsArg, datasetsArg, matchesPath)
	cmd := exec.Command(
		"python3", "exec_ops/unsupervised_reid/train.py",
		fmt.Sprintf("%d", e.dataset.ID), e.url, paramsArg, archArg, compsArg, datasetsArg, matchesPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	// add filename to the string dataset
	mydata := skyhook.StringData{Strings: []string{fmt.Sprintf("%d", e.dataset.ID)}}
	return exec_ops.WriteItem(e.url, e.dataset, "model", mydata)
}

func (e *TrainOp) Close() {}

func init() {
	skyhook.ExecOpImpls["unsupervised_reid"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("model"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			op := &TrainOp{
				url: url,
				node: node,
				dataset: node.OutputDatasets["model"],
			}
			return op, nil
		},
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
