package yolov3

import (
	"../../skyhook"
	"../../exec_ops"

	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Params struct {
	InputSize [2]int
	ConfigPath string
	ModelPath string
	MetaPath string

	ImageDatasetID int
	DetectionDatasetID int
}

func (p Params) GetConfigPath() string {
	if p.ConfigPath == "" {
		return "cfg/yolov3.cfg"
	} else {
		return p.ConfigPath
	}
}

func (p Params) GetModelPath() string {
	if p.ModelPath == "" {
		return "yolov3.weights"
	} else {
		return p.ModelPath
	}
}

func (p Params) GetMetaPath() string {
	if p.MetaPath == "" {
		return "cfg/coco.data"
	} else {
		return p.MetaPath
	}
}

func CreateParams(fname string, p Params, training bool) {
	// prepare configuration with this width/height
	configPath := p.GetConfigPath()
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join("darknet/", configPath)
	}
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "width=") && p.InputSize[0] > 0 {
			line = fmt.Sprintf("width=%d", p.InputSize[0])
		} else if strings.HasPrefix(line, "height=") && p.InputSize[1] > 0 {
			line = fmt.Sprintf("height=%d", p.InputSize[1])
		} else if training && strings.HasPrefix(line, "batch=") {
			line = "batch=64"
		} else if training && strings.HasPrefix(line, "subdivisions=") {
			line = "subdivisions=8"
		}
		file.Write([]byte(line+"\n"))
	}
	file.Close()
}

func Train(url string, node skyhook.TrainNode) error {
	var params Params
	skyhook.JsonUnmarshal([]byte(node.Params), &params)

	workingDir, err := os.Getwd()
	if err != nil {
		// shouldn't fail
		panic(err)
	}

	// create temporary directory for training config/example files
	log.Println("[yolov3-train] creating training and export directories")
	trainPath := filepath.Join(workingDir, "models", fmt.Sprintf("yolov3-%d", node.ID))
	if err := os.Mkdir(trainPath, 0755); err != nil {
		return fmt.Errorf("could not mkdir %s: %v", trainPath, err)
	}
	/*defer func() {
		os.RemoveAll(trainPath)
	}()*/

	exportPath := filepath.Join(os.TempDir(), fmt.Sprintf("yolov3-%d", node.ID))
	if err := os.Mkdir(exportPath, 0755); err != nil {
		return fmt.Errorf("could not mkdir %s: %v", exportPath, err)
	}
	defer func() {
		//os.RemoveAll(exportPath)
	}()

	// export the images and detections to a new folder in darknet format
	log.Println("[yolov3-train] exporting examples")
	datasets, err := exec_ops.GetDatasets(url, []int{params.ImageDatasetID, params.DetectionDatasetID})
	if err != nil {
		return err
	}
	items, err := exec_ops.GetItems(url, datasets)
	if err != nil {
		return err
	}
	counter := 0
	var imFnames []string
	for _, l := range items {
		counter += 1

		imData, err := l[0].LoadData()
		if err != nil {
			return err
		}
		imPath := filepath.Join(exportPath, fmt.Sprintf("%d.jpg", counter))
		file, err := os.Create(imPath)
		if err != nil {
			return err
		}
		imData.Encode("jpeg", file)
		file.Close()
		imFnames = append(imFnames, imPath)

		labelData, err := l[1].LoadData()
		if err != nil {
			return err
		}
		labelData_ := labelData.(skyhook.DetectionData)
		detections := labelData_.Detections[0]
		canvasDims := labelData_.Metadata.CanvasDims
		file, err = os.Create(filepath.Join(exportPath, fmt.Sprintf("%d.txt", counter)))
		if err != nil {
			return err
		}
		for _, detection := range detections {
			cx := float64(detection.Left+detection.Right)/2/float64(canvasDims[0])
			cy := float64(detection.Top+detection.Bottom)/2/float64(canvasDims[1])
			w := float64(detection.Right-detection.Left)/float64(canvasDims[0])
			h := float64(detection.Bottom-detection.Top)/float64(canvasDims[1])
			s := fmt.Sprintf("0 %v %v %v %v\n", cx, cy, w, h)
			file.Write([]byte(s))
		}
		file.Close()
	}

	log.Println("[yolov3-train] writing metadata files")
	// write the list of train/valid/test files
	rand.Shuffle(len(imFnames), func(i, j int) {
		imFnames[i], imFnames[j] = imFnames[j], imFnames[i]
	})
	numVal := len(imFnames)/10+1
	validFnames := strings.Join(imFnames[0:numVal], "\n") + "\n"
	trainFnames := strings.Join(imFnames[numVal:], "\n") + "\n"
	dsPaths := [3]string{
		filepath.Join(trainPath, "train.txt"),
		filepath.Join(trainPath, "valid.txt"),
		filepath.Join(trainPath, "test.txt"),
	}
	if err := ioutil.WriteFile(dsPaths[0], []byte(trainFnames), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile(dsPaths[1], []byte(validFnames), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile(dsPaths[2], []byte(validFnames), 0644); err != nil {
		return err
	}

	// yolov3.cfg
	yoloCfgPath := filepath.Join(trainPath, "yolov3.cfg")
	CreateParams(yoloCfgPath, params, true)

	// compute number of classes for obj.data/obj.names
	// it needs to match yolov3.cfg
	// TODO: we should actually:
	// (1) compute the # classes from the provided object detections
	// (2) write the .txt files according to those classes
	// (3) update yolov3.cfg filters/classes as needed
	bytes, err := ioutil.ReadFile(yoloCfgPath)
	if err != nil {
		return err
	}
	numClasses := 1
	for _, line := range strings.Split(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "classes=") && !strings.HasPrefix(line, "classes ") {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) < 2 {
			continue
		}
		numClasses, _ = strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
	}

	// obj.names
	var names []string
	for i := 0; i < numClasses; i++ {
		names = append(names, fmt.Sprintf("class%d", i))
	}
	namesPath := filepath.Join(trainPath, "obj.names")
	if err := ioutil.WriteFile(namesPath, []byte(strings.Join(names, "\n")), 0644); err != nil {
		return err
	}

	// obj.data
	objDataTmpl := `
classes=%d
train=%s
valid=%s
test=%s
names=%s
backup=%s
`
	objDataStr := fmt.Sprintf(objDataTmpl, numClasses, dsPaths[0], dsPaths[1], dsPaths[2], namesPath, exportPath)
	objDataPath := filepath.Join(trainPath, "obj.data")
	if err := ioutil.WriteFile(objDataPath, []byte(objDataStr), 0644); err != nil {
		return err
	}

	// run darknet job
	log.Println("[yolov3-train] begin training")
	cmd := exec.Command(
		"./darknet", "detector", "train", "-map",
		filepath.Join(trainPath, "obj.data"),
		filepath.Join(trainPath, "yolov3.cfg"),
		"darknet53.conv.74",
	)
	cmd.Dir = "darknet/"
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// parse stdout for mAP scores to determine when to stop training
	bestIterCh := make(chan int)
	go func() {
		rd := bufio.NewReader(stdout)
		var bestScore float64
		var bestAge int
		var bestIter int
		var curIter int
		for bestAge < 20 {
			line, err := rd.ReadString('\n')
			if err != nil {
				bestIter = -1
				break
			}
			log.Println("[yolov3-train] " + strings.TrimSpace(line))

			if strings.Contains(line, "mean average precision (mAP@0.50) = ") {
				line = strings.Split(line, "mean average precision (mAP@0.50) = ")[1]
				line = strings.Split(line, ",")[0]
				score := skyhook.ParseFloat(line)
				if score > bestScore {
					bestScore = score
					bestAge = 0
					bestIter = curIter
					log.Printf("[yolov3-train] got new best mAP %v @ iteration %v", bestScore, bestIter)
				} else {
					bestAge++
					log.Printf("[yolov3-train] %d iterations with bad mAP", bestAge)
				}
			}

			if strings.Contains(line, "next mAP calculation at ") {
				line = strings.Split(line, "next mAP calculation at ")[1]
				line = strings.Split(line, " ")[0]
				curIter = skyhook.ParseInt(line)
			}
		}
		cmd.Process.Kill()
		stdout.Close()
		bestIterCh <- bestIter
	}()
	cmd.Wait()
	bestIter := <- bestIterCh

	if bestIter == -1 {
		return fmt.Errorf("error running darknet")
	}

	skyhook.CopyFile(exportPath+"yolov3_best.weights", trainPath+"yolov3.weights")
	return nil
}

func Prepare(url string, trainNode skyhook.TrainNode, execNode skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func init() {
	skyhook.TrainOps["yolov3"] = skyhook.TrainOp{
		Requirements: func(url string, node skyhook.TrainNode) map[string]int {
			return map[string]int{}
		},
		Train: Train,
		Prepare: Prepare,
	}
}
