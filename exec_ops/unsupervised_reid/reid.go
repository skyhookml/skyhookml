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
	"strconv"
)

type Params struct {
	skyhook.PytorchTrainParams
	// MatchLengths should be TrackDuration/8, /4, /2, /1
	TrackDuration float64
}

// Return match lengths for PreprocessMatches based on framerate and p.TrackDuration.
func (p Params) GetMatchLengths(framerate [2]int) []int {
	var matchLengths []int
	for _, factor := range []float64{0.125, 0.25, 0.5, 1.0} {
		duration := p.TrackDuration * factor
		numFrames := int(duration * float64(framerate[0]) / float64(framerate[1]))
		matchLengths = append(matchLengths, numFrames)
	}
	return matchLengths
}

type TrainOp struct {
	url string
	node skyhook.Runnable
	dataset skyhook.Dataset
}

func (e *TrainOp) Parallelism() int {
	return 1
}

func (e *TrainOp) Apply(task skyhook.ExecTask) error {
	var params Params
	if err := exec_ops.DecodeParams(e.node, &params, false); err != nil {
		return err
	}
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
		os.RemoveAll(matchesPath)
	}()
	for _, l := range items {
		videoItem, detectionItem := l[0], l[1]
		key := detectionItem.Key

		// compute match lengths for this key based on the video framerate
		var videoMetadata skyhook.VideoMetadata
		skyhook.JsonUnmarshal([]byte(videoItem.Metadata), &videoMetadata)
		matchLengths := params.GetMatchLengths(videoMetadata.Framerate)

		log.Printf("[reid] pre-processing key %s with match_lengths=%v", key, matchLengths)
		labelData, err := detectionItem.LoadData()
		if err != nil {
			return fmt.Errorf("error loading label (detection) data: %v", err)
		}
		detections := labelData.(skyhook.DetectionData).Detections
		matches := PreprocessMatches(detections, matchLengths)
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

	e.dataset.Mkdir()

	paramsArg := e.node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	datasetsArg := string(skyhook.JsonMarshal(datasetList))
	fmt.Println(e.dataset.ID, e.url, paramsArg, archArg, compsArg, datasetsArg, matchesPath)
	cmd := exec.Command(
		"python3", "exec_ops/unsupervised_reid/train.py",
		strconv.Itoa(e.dataset.ID), e.url, paramsArg, archArg, compsArg, datasetsArg, matchesPath,
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

	// add to the file dataset
	fileMetadata := skyhook.FileMetadata{Filename: "model.pt"}
	_, err = exec_ops.AddItem(e.url, e.dataset, "model", "pt", "", string(skyhook.JsonMarshal(fileMetadata)))
	if err != nil {
		return err
	}

	return nil
}

func (e *TrainOp) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "unsupervised_reid",
			Name: "Unsupervised Re-identification",
			Description: "Self-Supervised Re-identification Model",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "video", DataTypes: []skyhook.DataType{skyhook.VideoType}},
			{Name: "detections", DataTypes: []skyhook.DataType{skyhook.DetectionType}},
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
				dataset: node.OutputDatasets["model"],
			}
			return op, nil
		},
		ImageName: "skyhookml/pytorch",
	})
}
