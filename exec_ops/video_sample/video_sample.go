package video_sample

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
)

type Params struct {
	Length int
	Count int

	// "random" or "uniform"
	Mode string
}

type VideoSample struct {
	URL string
	Params Params
	Datasets map[string]skyhook.Dataset
}

func (e *VideoSample) Parallelism() int {
	// each ffmpeg runs with two threads
	return runtime.NumCPU()/2
}

func (e *VideoSample) Apply(task skyhook.ExecTask) error {
	// Decode task metadata to get the samples we need to extract.
	var samples [][2]int
	skyhook.JsonUnmarshal([]byte(task.Metadata), &samples)

	log.Printf("extracting %d samples from %s", len(samples), task.Key)

	// Create map of where samples start.
	startToEnd := make(map[int][]int)
	for _, sample := range samples {
		startToEnd[sample[0]] = append(startToEnd[sample[0]], sample[1])
	}

	type ProcessingSample struct {
		Key string
		Start int
		End int
		Writers []skyhook.SequenceWriter
	}

	// Load input items, output datasets, and metadatas.
	var inputs []skyhook.Item
	var outputDatasets []skyhook.Dataset
	var metadatas []skyhook.DataMetadata

	inputs = append(inputs, task.Items["video"][0][0])
	outputDatasets = append(outputDatasets, e.Datasets["samples"])
	metadatas = append(metadatas, inputs[0].DecodeMetadata())

	for i, itemList := range task.Items["others"] {
		inputs = append(inputs, itemList[0])
		outputDatasets = append(outputDatasets, e.Datasets[fmt.Sprintf("others%d", i)])
		metadatas = append(metadatas, itemList[0].DecodeMetadata())
	}

	// Samples where we're currently in the middle of the intervals.
	processing := make(map[string]ProcessingSample)

	err := skyhook.PerFrame(inputs, func(pos int, datas []interface{}) error {
		// add segments that start at this frame to the processing set
		for _, end := range startToEnd[pos] {
			sampleKey := fmt.Sprintf("%s_%d_%d", task.Key, pos, end)
			if _, ok := processing[sampleKey]; ok {
				// duplicate interval
				continue
			}

			sample := ProcessingSample{
				Key: sampleKey,
				Start: pos,
				End: end,
				Writers: make([]skyhook.SequenceWriter, len(inputs)),
			}

			for i, ds := range outputDatasets {
				// Add an item to the dataset first.
				// To do so, we need to know the ext/format/metadata.
				// If input/output type match, then we can copy it from the input.
				// If they don't match (video input, image output), we handle the special case.
				var ext, format string
				var metadata skyhook.DataMetadata
				if inputs[i].Dataset.DataType == skyhook.VideoType && ds.DataType == skyhook.ImageType {
					ext = "jpg"
					format = "jpeg"
					metadata = skyhook.NoMetadata{}
				} else {
					metadata = metadatas[i]
					ext, format = inputs[i].DataSpec().GetDefaultExtAndFormat(datas[i], metadatas[i])
				}
				if ds.DataType == skyhook.VideoType {
					// For video outputs, update the Duration of the metadata so that it matches the sample duration.
					vmeta := metadata.(skyhook.VideoMetadata)
					vmeta.Duration = float64((end-pos)*vmeta.Framerate[1])/float64(vmeta.Framerate[0])
					metadata = vmeta
				}
				item, err := exec_ops.AddItem(e.URL, ds, sampleKey, ext, format, metadata)
				if err != nil {
					return err
				}
				sample.Writers[i] = item.LoadWriter()
			}

			processing[sampleKey] = sample
		}

		// push data onto processing segments
		for _, sample := range processing {
			for i := range datas {
				err := sample.Writers[i].Write(datas[i])
				if err != nil {
					return err
				}
			}
		}

		// remove processing segments that end here
		for sampleKey, sample := range processing {
			if pos+1 < sample.End {
				continue
			}
			delete(processing, sampleKey)
			// Close the writers, and return if we encounter an error.
			// But we have to be a bit careful to always close all the writers here,
			// now that we've removed the sample from the processing set.
			var closeErr error
			for _, writer := range sample.Writers {
				err := writer.Close()
				if err != nil {
					closeErr = err
				}
			}
			if closeErr != nil {
				return closeErr
			}
		}

		return nil
	})

	// Before checking err, finish closing any remaining writers.
	// On a successful run, they should've all been closed already.
	for sampleKey, sample := range processing {
		delete(processing, sampleKey)
		for _, writer := range sample.Writers {
			writer.Close()
		}
		if err == nil {
			err = fmt.Errorf("segment still processing after iteration")
		}
	}

	return err
}

func (e *VideoSample) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "video_sample",
			Name: "Sample video",
			Description: "Sample images or segments from video",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "video", DataTypes: []skyhook.DataType{skyhook.VideoType}},
			{Name: "others", Variable: true},
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured: %v", err)
			}

			groupedItems := exec_ops.GroupItems(allItems)

			// only keep items that have length set, and at least params.Length
			type Item struct {
				Item skyhook.Item
				Metadata skyhook.VideoMetadata
				NumFrames int
			}
			var videoItems []Item
			for _, group := range groupedItems {
				item := group["video"][0]
				metadata := item.DecodeMetadata().(skyhook.VideoMetadata)

				// estimate num frames from framerate and duration
				numFrames := int(metadata.Duration * float64(metadata.Framerate[0])) / metadata.Framerate[1]
				if numFrames < params.Length {
					continue
				}
				videoItems = append(videoItems, Item{item, metadata, numFrames})
			}

			// select the samples
			samples := make(map[string][][2]int)
			// TODO: uniform not implemented yet, so we just silently do random
			if params.Mode == "random" || params.Mode == "uniform" {
				// sample item based on how many possible segments there are in the item
				// (which depends on item and segment lengths)
				weights := make([]int, len(videoItems))
				var sum int
				for i, item := range videoItems {
					weight := item.NumFrames - params.Length
					weights[i] = weight
					sum += weight
				}
				sampleItem := func() Item {
					r := rand.Intn(sum)
					for i, w := range weights {
						r -= w
						if r < 0 {
							return videoItems[i]
						}
					}
					return videoItems[len(videoItems)-1]
				}

				// sample iteratively
				for i := 0; i < params.Count; i++ {
					item := sampleItem()
					startIdx := rand.Intn(item.NumFrames - params.Length + 1)
					samples[item.Item.Key] = append(samples[item.Item.Key], [2]int{startIdx, startIdx+params.Length})
				}
			} else {
				return nil, fmt.Errorf("unknown video_sample mode %s", params.Mode)
			}

			var tasks []skyhook.ExecTask
			for key, intervals := range samples {
				// collect items for this task
				curItems := make(map[string][][]skyhook.Item)
				for name, itemList := range groupedItems[key] {
					curItems[name] = make([][]skyhook.Item, len(itemList))
					for i, item := range itemList {
						curItems[name][i] = []skyhook.Item{item}
					}
				}
				tasks = append(tasks, skyhook.ExecTask{
					Key: key,
					Items: curItems,
					Metadata: string(skyhook.JsonMarshal(intervals)),
				})
			}
			return tasks, nil
		},
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return nil, err
			}
			op := &VideoSample{
				URL: url,
				Params: params,
				Datasets: node.OutputDatasets,
			}
			return op, nil
		},
		GetOutputs: func(rawParams string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			// we always output samples, which is image if params.Length == 1 and video otherwise
			// but then output others0, others1, ... for each others input (which borrows type from its input)

			var params Params
			err := json.Unmarshal([]byte(rawParams), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				// so we leave it unchanged
				return nil
			}

			// If input is neither video nor image, we copy the input type.
			// For video, it is video output unless sample length = 1 in which case it's image.
			// Image input always yields image output.
			getOutputType := func(inputType skyhook.DataType) skyhook.DataType {
				if inputType != skyhook.VideoType {
					return inputType
				}
				if params.Length == 1 {
					return skyhook.ImageType
				} else {
					return skyhook.VideoType
				}
			}

			// first add samples type (based on whether video input is image or video)
			if len(inputTypes["video"]) == 0 {
				return nil
			}
			outputs := []skyhook.ExecOutput{{
				Name: "samples",
				DataType: getOutputType(inputTypes["video"][0]),
			}}

			// now add others, which copies the type of each one
			for i, inputType := range inputTypes["others"] {
				outputs = append(outputs, skyhook.ExecOutput{
					Name: fmt.Sprintf("others%d", i),
					DataType: getOutputType(inputType),
				})
			}
			return outputs
		},
		ImageName: "skyhookml/basic",
	})
}
