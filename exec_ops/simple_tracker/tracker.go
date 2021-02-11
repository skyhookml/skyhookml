package simple_tracker

import (
	"../../skyhook"
	"../../exec_ops"

	"runtime"

	"github.com/mitroadmaps/gomapinfer/common"
)

type Tracker struct {
	URL string
	Node skyhook.ExecNode
	Dataset skyhook.Dataset
}

type TrackedDetection struct {
	skyhook.Detection
	FrameIdx int
}

func (d TrackedDetection) Rectangle() common.Rectangle {
	return common.Rectangle{
		common.Point{float64(d.Left), float64(d.Top)},
		common.Point{float64(d.Right), float64(d.Bottom)},
	}
}

func (e *Tracker) Parallelism() int {
	return runtime.NumCPU()
}

func (e *Tracker) Apply(task skyhook.ExecTask) error {
	data, err := task.Items[0].LoadData()
	if err != nil {
		return err
	}
	detectionData := data.(skyhook.DetectionData)
	detections := detectionData.Detections

	ndetections := make([][]skyhook.Detection, len(detections))
	activeTracks := make(map[int][]TrackedDetection)
	nextTrackID := 1

	abs := func(x int) int {
		if x < 0 {
			return -x
		} else {
			return x
		}
	}

	// helper function: estimate current position of track in new frame
	// we make the estimation using the object's recent average speed
	// TODO: parameterize targetFrame
	estimatePosition := func(curFrame int, track []TrackedDetection) TrackedDetection {
		lastDetection := track[len(track)-1]

		if len(track) == 1 {
			return lastDetection
		}

		// find detection closest to a frame a certain interval in the past
		// use this to get a speed estimate
		targetFrame := lastDetection.FrameIdx - 5
		var bestDetection TrackedDetection
		var bestOffset int = -1
		for _, d := range track[0:len(track)-1] {
			offset := abs(d.FrameIdx - targetFrame)
			if bestOffset == -1 || offset < bestOffset {
				bestDetection = d
				bestOffset = offset
			}
		}
		dx := float64(lastDetection.Left+lastDetection.Right-bestDetection.Left-bestDetection.Right)/2
		dy := float64(lastDetection.Top+lastDetection.Bottom-bestDetection.Top-bestDetection.Bottom)/2
		scale := float64(curFrame - lastDetection.FrameIdx)/float64(lastDetection.FrameIdx-bestDetection.FrameIdx)
		motion := [2]int{int(dx*scale), int(dy*scale)}

		return TrackedDetection{Detection: skyhook.Detection{
			Left: lastDetection.Left + motion[0],
			Top: lastDetection.Top + motion[1],
			Right: lastDetection.Right + motion[0],
			Bottom: lastDetection.Bottom + motion[1],
		}}
	}

	// helper function: extract matches from matrix
	// I don't think hungarian algorithm works too well here, instead we do simple greedy approach
	extractMatches := func(matrix [][]float64) [][2]int {
		if len(matrix) == 0 || len(matrix[0]) == 0 {
			return nil
		}

		// get max probability and index over columns along each row
		type Candidate struct {
			Row int
			Col int
			Score float64
		}
		rowMax := make([]Candidate, len(matrix))
		for i := range matrix {
			// TODO: make the 0.1 minimum IOU score a parameter (?)
			rowMax[i] = Candidate{-1, -1, 0.1}
			for j := range matrix[i] {
				prob := matrix[i][j]
				if prob > rowMax[i].Score {
					rowMax[i] = Candidate{i, j, prob}
				}
			}
		}

		// now make sure each row picked a unique column
		// in cases of conflicts, resolve via max probability
		// the losing row in the conflict would then match to nothing
		colMax := make([]Candidate, len(matrix[0]))
		for i := 0; i < len(matrix[0]); i++ {
			colMax[i] = Candidate{-1, -1, 0}
		}
		for _, candidate := range rowMax {
			if candidate.Col == -1 {
				continue
			}
			if candidate.Score > colMax[candidate.Col].Score {
				colMax[candidate.Col] = candidate
			}
		}

		// finally we can enumerate the matches
		var matches [][2]int
		for _, candidate := range colMax {
			if candidate.Col == -1 {
				continue
			}
			matches = append(matches, [2]int{candidate.Row, candidate.Col})
		}
		return matches
	}

	for frameIdx, dlist := range detections {
		// get matrix matching active tracks to new detections
		// rows: active tracks
		// cols: current detections
		// values: IoU score between the estimated position of track in current frame, and detection
		var activeIDs []int
		for id := range activeTracks {
			activeIDs = append(activeIDs, id)
		}

		matrix := make([][]float64, len(activeIDs))
		for i, trackID := range activeIDs {
			matrix[i] = make([]float64, len(dlist))
			curEstimate := estimatePosition(frameIdx, activeTracks[trackID])
			trackRect := curEstimate.Rectangle()

			for j, detection := range dlist {
				detRect := TrackedDetection{Detection: detection}.Rectangle()
				matrix[i][j] = trackRect.IOU(detRect)
			}
		}

		// compute matches, and add detections to the matched tracks
		// detections that didn't match to any track will form new tracks
		matches := extractMatches(matrix)
		matchedDetections := make([]bool, len(dlist))
		for _, match := range matches {
			trackIdx, detectionIdx := match[0], match[1]
			trackID := activeIDs[trackIdx]
			detection := dlist[detectionIdx]
			detection.TrackID = trackID
			activeTracks[trackID] = append(activeTracks[trackID], TrackedDetection{
				Detection: detection,
				FrameIdx: frameIdx,
			})
			ndetections[frameIdx] = append(ndetections[frameIdx], detection)
			matchedDetections[detectionIdx] = true
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
	}

	outputData := skyhook.DetectionData{
		Detections: ndetections,
		Metadata: detectionData.Metadata,
	}
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, outputData)
}

func (e *Tracker) Close() {}

func init() {
	skyhook.ExecOpImpls["simple_tracker"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, error) {
			op := &Tracker{url, node, outputDatasets[0]}
			return op, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
