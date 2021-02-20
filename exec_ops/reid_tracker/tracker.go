package reid_tracker

import (
	"../../skyhook"
	"../../exec_ops"
	strack "../../exec_ops/simple_tracker"
	"../../exec_ops/pytorch"

	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"sync"
)

const MinPadding = 4
const CropSize = 64

type TrackedDetection struct {
	skyhook.Detection
	FrameIdx int
	Image skyhook.Image
}

type Tracker struct {
	URL string
	Dataset skyhook.Dataset

	mu sync.Mutex
	cmd *skyhook.Cmd
	stdin io.WriteCloser
	rd *bufio.Reader
}

func Prepare(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
	arch, components, _, err := pytorch.GetArgs(url, node)
	if err != nil {
		return nil, err
	}

	// get the model path from the first input dataset
	datasets, err := exec_ops.ParentsToDatasets(url, node.Parents[0:1])
	if err != nil {
		return nil, err
	}
	modelItems, err := exec_ops.GetItems(url, datasets)
	if err != nil {
		return nil, err
	}
	modelItem := modelItems["model"][0]
	strdata, err := modelItem.LoadData()
	if err != nil {
		return nil, err
	}
	modelPath := strdata.(skyhook.StringData).Strings[0]

	paramsArg := node.Params
	archArg := string(skyhook.JsonMarshal(arch))
	compsArg := string(skyhook.JsonMarshal(components))
	cmd := skyhook.Command(
		fmt.Sprintf("reid_tracker-%s", node.Name), skyhook.CommandOptions{},
		"python3", "exec_ops/reid_tracker/run.py",
		modelPath, paramsArg, archArg, compsArg,
	)

	return &Tracker{
		URL: url,
		Dataset: outputDatasets[0],
		cmd: cmd,
		stdin: cmd.Stdin(),
		rd: bufio.NewReader(cmd.Stdout()),
	}, nil
}

func (e *Tracker) Parallelism() int {
	return runtime.NumCPU()
}

func (e *Tracker) Apply(task skyhook.ExecTask) error {
	videoData, err := task.Items[0].LoadData()
	if err != nil {
		return err
	}

	data1, err := task.Items[1].LoadData()
	if err != nil {
		return err
	}
	detectionData := data1.(skyhook.DetectionData)

	ndetections := make([][]skyhook.Detection, len(detectionData.Detections))
	activeTracks := make(map[int][]TrackedDetection)
	nextTrackID := 1

	datas := []skyhook.Data{videoData, detectionData}
	err = skyhook.PerFrame(datas, func(frameIdx int, datas []skyhook.Data) error {
		im := datas[0].(skyhook.ImageData).Images[0]
		detectionData := datas[1].(skyhook.DetectionData)
		detectionDims := detectionData.Metadata.CanvasDims
		dlist := detectionData.Detections[0]

		// prepare query to python script:
		// (1) batch of images from tracks
		// (2) images of detections in current frame
		var leftImages []skyhook.Image
		var rightImages []skyhook.Image

		var activeIDs []int
		for id, track := range activeTracks {
			activeIDs = append(activeIDs, id)
			leftImages = append(leftImages, track[len(track)-1].Image)
		}

		for _, d := range dlist {
			sx := skyhook.Clip(d.Left * im.Width / detectionDims[0], 0, im.Width-MinPadding)
			ex := skyhook.Clip(d.Right * im.Width / detectionDims[0], sx+MinPadding, im.Width)
			sy := skyhook.Clip(d.Top * im.Height / detectionDims[1], 0, im.Height-MinPadding)
			ey := skyhook.Clip(d.Bottom * im.Height / detectionDims[1], sy+MinPadding, im.Height)
			crop := im.Crop(sx, sy, ex, ey)

			// resize to max 64x64 side
			factor := math.Min(CropSize/float64(crop.Width), CropSize/float64(crop.Height))
			resizeWidth := skyhook.Clip(int(factor*float64(crop.Width)), MinPadding, CropSize)
			resizeHeight := skyhook.Clip(int(factor*float64(crop.Height)), MinPadding, CropSize)
			resized := crop.Resize(resizeWidth, resizeHeight)
			fix := skyhook.NewImage(CropSize, CropSize)
			fix.DrawImage(0, 0, resized)
			rightImages = append(rightImages, fix)
		}

		matchedDetections := make([]bool, len(dlist))
		if len(dlist) > 0 && len(activeTracks) > 0 {
			e.mu.Lock()
			header := make([]byte, 8)
			binary.BigEndian.PutUint32(header[0:4], uint32(len(activeTracks)))
			binary.BigEndian.PutUint32(header[4:8], uint32(len(dlist)))
			e.stdin.Write(header)
			for _, im := range leftImages {
				e.stdin.Write(im.Bytes)
			}
			for _, im := range rightImages {
				e.stdin.Write(im.Bytes)
			}

			signature := "json"
			var line string
			var err error
			for {
				line, err = e.rd.ReadString('\n')
				if err != nil || strings.Contains(line, signature) {
					break
				}
			}
			e.mu.Unlock()

			if err != nil {
				return fmt.Errorf("error reading from reid script: %v", err)
			}

			line = strings.TrimSpace(line[len(signature):])
			var matrix [][]float64
			skyhook.JsonUnmarshal([]byte(line), &matrix)
			matches := strack.ExtractMatches(matrix)

			for _, match := range matches {
				trackIdx, detectionIdx := match[0], match[1]
				trackID := activeIDs[trackIdx]
				detection := dlist[detectionIdx]
				detection.TrackID = trackID
				activeTracks[trackID] = append(activeTracks[trackID], TrackedDetection{
					Detection: detection,
					FrameIdx: frameIdx,
					Image: rightImages[detectionIdx],
				})
				ndetections[frameIdx] = append(ndetections[frameIdx], detection)
				matchedDetections[detectionIdx] = true
			}
		}

		for j, detection := range dlist {
			if matchedDetections[j] {
				continue
			}
			trackID := nextTrackID
			nextTrackID++
			detection.TrackID = trackID
			activeTracks[trackID] = []TrackedDetection{TrackedDetection{
				Detection: detection,
				FrameIdx: frameIdx,
				Image: rightImages[j],
			}}
			ndetections[frameIdx] = append(ndetections[frameIdx], detection)
		}

		// remove old active tracks
		for trackID, track := range activeTracks {
			// TODO: parameter
			if frameIdx - track[len(track)-1].FrameIdx < 10 {
				continue
			}
			delete(activeTracks, trackID)
		}

		return nil
	})
	if err != nil {
		return err
	}

	outputData := skyhook.DetectionData{
		Detections: ndetections,
		Metadata: detectionData.Metadata,
	}
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, outputData)
}

func (e *Tracker) Close() {}

func init() {
	skyhook.ExecOpImpls["reid_tracker"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: func(url string, node skyhook.ExecNode, rawItems [][]skyhook.Item) ([]skyhook.ExecTask, error) {
			// the first input dataset in the model
			// so we just provide the rest to SimpleTasks
			return exec_ops.SimpleTasks(url, node, rawItems[1:])
		},
		Prepare: Prepare,
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/pytorch", nil
		},
	}
}
