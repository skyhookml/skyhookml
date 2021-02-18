package reid

import (
	"../../skyhook"

	"runtime"

	"github.com/mitroadmaps/gomapinfer/common"
)

const MaxMatchAge = 10
var MatchLengths = []int{2, 4, 8, 16, 32, 64}
var MaxMatchLength = MatchLengths[len(MatchLengths) - 1]

// Matches detections in a given frame forwards through frames to the MaxMatchLength.
// Returns a map detectionIdx (in startFrame) -> set of (frame, detectionIdx in frame)
func matchFrom(detections [][]skyhook.Detection, startFrame int) map[int]map[[2]int]bool {
	toRect := func(d skyhook.Detection) common.Rectangle {
		return common.Rectangle{
			common.Point{float64(d.Left), float64(d.Top)},
			common.Point{float64(d.Right), float64(d.Bottom)},
		}
	}

	// from detection_idx in startFrame to list of matching tuples (frame idx, detection idx)
	curMatches := make(map[int]map[[2]int]bool)
	finalMatches := make(map[int]map[[2]int]bool)
	for idx := range detections[startFrame] {
		curMatches[idx] = make(map[[2]int]bool)
		finalMatches[idx] = make(map[[2]int]bool)
		curMatches[idx][[2]int{startFrame, idx}] = true
		finalMatches[idx][[2]int{startFrame, idx}] = true
	}

	for frameIdx := startFrame+1; frameIdx <= startFrame+MaxMatchLength && frameIdx < len(detections); frameIdx++ {
		// find the detections we need to match
		checkSet := make(map[[2]int]bool)
		for _, matches := range curMatches {
			for t := range matches {
				if frameIdx - t[0] >= MaxMatchAge {
					delete(matches, t)
					continue
				}
				checkSet[t] = true
			}
		}

		// determine connections between those in checkSet and those in current frame
		connections := make(map[[2]int][][2]int)
		for left := range checkSet {
			leftFrame, leftIdx := left[0], left[1]
			for rightIdx := 0; rightIdx < len(detections[frameIdx]); rightIdx++ {
				leftRect := toRect(detections[leftFrame][leftIdx])
				rightRect := toRect(detections[frameIdx][rightIdx])
				if leftRect.IOU(rightRect) < 0.1 {
					continue
				}
				connections[left] = append(connections[left], [2]int{frameIdx, rightIdx})
			}
		}

		for idx, matches := range curMatches {
			for left := range matches {
				for _, right := range connections[left] {
					matches[right] = true
					finalMatches[idx][right] = true
				}
			}
		}
	}

	return finalMatches
}

// Computes plausible matchings between every detection and other detections forwards/backwards in time.
// Returns map from (frame idx, detection idx, match length) -> list of plausible detection idx in (frame idx + match length)
func PreprocessMatches(detections [][]skyhook.Detection) map[[3]int][]int {
	matches := make(map[[3]int][]int)
	matchLengthSet := make(map[int]bool)
	for _, matchLength := range MatchLengths {
		matchLengthSet[matchLength] = true
	}

	// multi-threaded pre-processing
	// we process each start frame independently
	nthreads := runtime.NumCPU()
	ch := make(chan int)
	donech := make(chan map[[3]int][]int)
	for i := 0; i < nthreads; i++ {
		go func() {
			threadMatches := make(map[[3]int][]int)
			for startFrame := range ch {
				frameMatches := matchFrom(detections, startFrame)
				for curIdx := range frameMatches {
					for right := range frameMatches[curIdx] {
						matchLength := right[0] - startFrame
						rightIdx := right[1]
						if !matchLengthSet[matchLength] {
							continue
						}
						threadMatches[[3]int{startFrame, curIdx, matchLength}] = append(threadMatches[[3]int{startFrame, curIdx, matchLength}], rightIdx)
					}
				}
			}
			donech <- threadMatches
		}()
	}
	for startFrame := 0; startFrame < len(detections); startFrame++ {
		ch <- startFrame
	}
	close(ch)
	for i := 0; i < nthreads; i++ {
		threadMatches := <- donech
		for k, v := range threadMatches {
			matches[k] = v
		}
	}

	return matches
}
