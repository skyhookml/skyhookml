package yolov3

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

func Prepare(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
	var params Params
	skyhook.JsonUnmarshal([]byte(node.Params), &params)

	// load model path from first input dataset
	dataset := node.InputDatasets["model"][0]
	modelItems, err := exec_ops.GetDatasetItems(url, dataset)
	if err != nil {
		return nil, err
	}
	strdata, err := modelItems["model"].LoadData()
	if err != nil {
		return nil, err
	}
	modelPath := strdata.(skyhook.StringData).Strings[0]

	// load category names
	var categories []string
	trainPath := fmt.Sprintf("models/yolov3-%s/", modelPath)
	bytes, err := ioutil.ReadFile(filepath.Join(trainPath, "obj.names"))
	if err != nil {
		return nil, fmt.Errorf("error reading obj.names: %v", err)
	}
	for _, line := range strings.Split(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		categories = append(categories, line)
	}

	// run yolov3 script
	batchSize := 8
	cmd := skyhook.Command(
		"yolov3-exec", skyhook.CommandOptions{},
		"python3", "exec_ops/yolov3/run.py",
		modelPath,
		fmt.Sprintf("%d", batchSize),
		fmt.Sprintf("%d", params.InputSize[0]), fmt.Sprintf("%d", params.InputSize[1]),
	)

	return &Yolov3{
		URL: url,
		Dataset: node.OutputDatasets["detections"],
		cmd: cmd,
		stdin: cmd.Stdin(),
		rd: bufio.NewReader(cmd.Stdout()),
		batchSize: batchSize,
		dims: params.InputSize,
		categories: categories,
	}, nil
}

type Yolov3 struct {
	URL string
	Dataset skyhook.Dataset

	mu sync.Mutex
	cmd *skyhook.Cmd
	stdin io.WriteCloser
	rd *bufio.Reader
	batchSize int
	dims [2]int
	categories []string
}

func (e *Yolov3) Parallelism() int {
	return 1
}

func (e *Yolov3) Apply(task skyhook.ExecTask) error {
	data, err := task.Items["images"][0][0].LoadData()
	if err != nil {
		return err
	}
	reader := data.(skyhook.ReadableData).Reader()
	defer reader.Close()
	var detections [][]skyhook.Detection
	zeroImage := skyhook.NewImage(e.dims[0], e.dims[1])
	for {
		imageData, err := reader.Read(e.batchSize)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		images := imageData.(skyhook.ImageData).Images

		e.mu.Lock()
		// write this batch of images
		for _, im := range images {
			if im.Width != e.dims[0] || im.Height != e.dims[1] {
				im = im.Resize(e.dims[0], e.dims[1])
			}
			e.stdin.Write(im.Bytes)
		}
		for i := len(images); i < e.batchSize; i++ {
			e.stdin.Write(zeroImage.Bytes)
		}

		// read the output detections for the batch
		signature := "json"
		var line string
		for {
			line, err = e.rd.ReadString('\n')
			if err != nil || strings.Contains(line, signature) {
				break
			}
		}
		e.mu.Unlock()

		if err != nil {
			return fmt.Errorf("error reading from yolov3 script: %v", err)
		}

		line = strings.TrimSpace(line[len(signature):])
		var batchDetections [][]skyhook.Detection
		skyhook.JsonUnmarshal([]byte(line), &batchDetections)
		detections = append(detections, batchDetections[0:len(images)]...)
	}

	output := skyhook.DetectionData{
		Detections: detections,
		Metadata: skyhook.DetectionMetadata{
			CanvasDims: e.dims,
			Categories: e.categories,
		},
	}
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, output)
}

func (e *Yolov3) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stdin.Close()
	if e.cmd != nil {
		e.cmd.Wait()
		e.cmd = nil
	}
}

func init() {
	skyhook.ExecOpImpls["yolov3_infer"] = skyhook.ExecOpImpl{
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, rawItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// we want to use only images for SimpleTasks, not model
			return exec_ops.SimpleTasks(node, map[string][][]skyhook.Item{"images": rawItems["images"]})
		},
		Prepare: Prepare,
		Incremental: true,
		GetOutputKeys: func(node skyhook.ExecNode, inputs map[string][][]string) []string {
			inputs = map[string][][]string{"images": inputs["images"]}
			return exec_ops.MapGetOutputKeys(node, inputs)
		},
		GetNeededInputs: func(node skyhook.ExecNode, outputs []string) map[string][][]string {
			neededInputs := exec_ops.MapGetNeededInputs(node, outputs)
			neededInputs["model"] = [][]string{{"model"}}
			return neededInputs
		},
		ImageName: func(node skyhook.Runnable) (string, error) {
			return "skyhookml/yolov3", nil
		},
	}
}
