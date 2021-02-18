package reid

import (
	"../../skyhook"
	"../../exec_ops"
	"../../train_ops/pytorch"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func Train(url string, node skyhook.TrainNode) error {
	arch, components, datasets, err := pytorch.GetArgs(url, node)
	if err != nil {
		return err
	}

	// pre-process the detections
	var videoDataset, detectionDataset *skyhook.Dataset
	for _, ds := range datasets {
		if ds.DataType == skyhook.VideoType {
			videoDataset = ds
		} else if ds.DataType == skyhook.DetectionType {
			detectionDataset = ds
		}
	}
	items, err := exec_ops.GetItems(url, []skyhook.Dataset{*videoDataset, *detectionDataset})
	if err != nil {
		return err
	}
	matchesPath := filepath.Join(os.TempDir(), fmt.Sprintf("reid-%d", node.ID))
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

	paramsArg := node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasets))
	fmt.Println(node.ID, url, paramsArg, archArg, compsArg, datasetsArg, matchesPath)
	cmd := exec.Command(
		"python3", "train_ops/unsupervised_reid/train.py",
		fmt.Sprintf("%d", node.ID), url, paramsArg, archArg, compsArg, datasetsArg, matchesPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func Prepare(url string, trainNode skyhook.TrainNode, execNode skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
	return nil, nil
}

func init() {
	skyhook.TrainOps["unsupervised_reid"] = skyhook.TrainOp{
		Requirements: func(url string, node skyhook.TrainNode) map[string]int {
			return map[string]int{}
		},
		Train: Train,
		Prepare: Prepare,
		ImageName: func(url string, node skyhook.TrainNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
