package video_sample

import (
	"../../skyhook"
	"../../exec_ops"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Node skyhook.ExecNode
	Params Params
	Datasets map[string]skyhook.Dataset
}

func (e *VideoSample) Parallelism() int {
	// each ffmpeg runs with two threads
	return runtime.NumCPU()/2
}

func (e *VideoSample) Apply(task skyhook.ExecTask) error {
	// decode task metadata to get the samples we need to extract
	var samples [][2]int
	skyhook.JsonUnmarshal([]byte(task.Metadata), &samples)

	log.Printf("[video_sample %s] extracting %d samples from %s", e.Node.Name, len(samples), task.Key)

	processVideo := func(vdata skyhook.VideoData) (map[string]skyhook.Data, error) {
		// create map of where samples start
		startToEnd := make(map[int][]int)
		for _, sample := range samples {
			startToEnd[sample[0]] = append(startToEnd[sample[0]], sample[1])
		}

		outputs := make(map[string]skyhook.Data)

		type ProcessingClip struct {
			Key string
			Start int
			End int
			Ch chan skyhook.Image
		}
		type PendingResponse struct {
			Key string
			Data skyhook.VideoData
			Error error
		}
		// segments where we're currently in the middle of the intervals
		processing := make(map[string]ProcessingClip)
		// segments that we finished providing input, and just need to wait for the encoded video bytes
		pending := make(map[string]bool)
		ch := make(chan PendingResponse)

		counter := 0
		err := vdata.Iterator().Iterate(32, func(im skyhook.Image) {
			// add segments that start at this frame to the processing set
			for _, end := range startToEnd[counter] {
				sampleKey := fmt.Sprintf("%s_%d_%d", task.Key, counter, end)
				if _, ok := processing[sampleKey]; ok {
					// duplicate interval
					continue
				}

				// for images, we can do it quickly
				if end - counter == 1 {
					outputs[sampleKey] = skyhook.ImageData{Images: []skyhook.Image{im}}
					continue
				}

				pc := ProcessingClip{
					Key: sampleKey,
					Start: counter,
					End: end,
					Ch: make(chan skyhook.Image),
				}
				processing[sampleKey] = pc
				pending[sampleKey] = true
				go func() {
					r, cmd := skyhook.MakeVideo(&skyhook.ChanReader{pc.Ch}, vdata.Metadata.Dims, vdata.Metadata.Framerate)
					buf := new(bytes.Buffer)
					_, err := io.Copy(buf, r)
					if err != nil {
						r.Close()
						cmd.Wait()
						ch <- PendingResponse{sampleKey, skyhook.VideoData{}, err}
						return
					}
					r.Close()
					cmd.Wait()
					sampleMeta := skyhook.VideoMetadata{
						Dims: vdata.Metadata.Dims,
						Framerate: vdata.Metadata.Framerate,
						Duration: float64((pc.End-pc.Start)*vdata.Metadata.Framerate[1])/float64(vdata.Metadata.Framerate[0]),
					}
					sampleData := skyhook.VideoData{
						Metadata: sampleMeta,
						Bytes: buf.Bytes(),
					}
					ch <- PendingResponse{sampleKey, sampleData, nil}
				}()
			}

			// push image onto processing segments
			for _, pc := range processing {
				pc.Ch <- im
			}

			counter++

			// remove processing segments that end here
			for sampleKey, pc := range processing {
				if counter < pc.End {
					continue
				}
				close(pc.Ch)
				delete(processing, sampleKey)
			}
		})

		// before checking err, finish getting all the pending responses
		for sampleKey, pc := range processing {
			close(pc.Ch)
			delete(processing, sampleKey)
			if err == nil {
				err = fmt.Errorf("segment still processing after iteration")
			}
		}
		for len(pending) > 0 {
			resp := <- ch
			delete(pending, resp.Key)
			if resp.Error == nil {
				outputs[resp.Key] = resp.Data
			} else if err == nil {
				err = resp.Error
			}
		}

		return outputs, err
	}

	process := func(item skyhook.Item) (map[string]skyhook.Data, error) {
		data, err := item.LoadData()
		if err != nil {
			return nil, err
		}

		if data.Type() == skyhook.VideoType {
			return processVideo(data.(skyhook.VideoData))
		}

		// if this isn't video, then we currently assume we can slice the data directly
		sliceData := data.(skyhook.SliceData)
		outputs := make(map[string]skyhook.Data)
		for _, sample := range samples {
			sampleKey := fmt.Sprintf("%s_%d_%d", task.Key, sample[0], sample[1])
			sampleData := sliceData.Slice(sample[0], sample[1])
			outputs[sampleKey] = sampleData
		}
		return outputs, nil
	}

	processAndWrite := func(item skyhook.Item, dataset skyhook.Dataset) error {
		outputs, err := process(item)
		if err != nil {
			return err
		}
		for key, data := range outputs {
			err := exec_ops.WriteItem(e.URL, dataset, key, data)
			if err != nil {
				return fmt.Errorf("error writing item to dataset %d: %v", dataset.ID, err)
			}
		}
		return nil
	}

	err := processAndWrite(task.Items["video"][0][0], e.Datasets["samples"])
	if err != nil {
		return err
	}
	for i, itemList := range task.Items["others"] {
		err := processAndWrite(itemList[0], e.Datasets[fmt.Sprintf("others%d", i)])
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *VideoSample) Close() {}

func init() {
	skyhook.ExecOpImpls["video_sample"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: func(url string, node skyhook.ExecNode, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured", err)
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
				var metadata skyhook.VideoMetadata
				err := json.Unmarshal([]byte(item.Metadata), &metadata)
				if err != nil {
					continue
				}

				// estimate num frames from framerate and duration
				numFrames := int(metadata.Duration * float64(metadata.Framerate[0])) / metadata.Framerate[1]
				if numFrames < params.Length {
					continue
				}
				videoItems = append(videoItems, Item{item, metadata, numFrames})
			}

			// select the samples
			samples := make(map[string][][2]int)
			if params.Mode == "random" {
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
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured", err)
			}
			op := &VideoSample{
				URL: url,
				Node: node,
				Params: params,
				Datasets: outputDatasets,
			}
			return op, nil
		},
		GetOutputs: func(url string, node skyhook.ExecNode) []skyhook.ExecOutput {
			// we always output samples, which is image if params.Length == 1 and video otherwise
			// but then output others0, others1, ... for each others input (which borrows type from its input)

			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				// so we leave it unchanged
				return node.Outputs
			}

			// return empty string on error
			getOutputType := func(parent skyhook.ExecParent) skyhook.DataType {
				inputType, err := exec_ops.ParentToDataType(url, parent)
				if err != nil {
					log.Printf("[render] warning: unable to compute outputs: %v", err)
					return ""
				}
				if inputType != skyhook.VideoType {
					return inputType
				}
				if params.Length == 1 {
					return skyhook.ImageType
				} else {
					return skyhook.VideoType
				}
			}

			parents := node.GetParents()
			if len(parents["video"]) == 0 {
				return node.Outputs
			}
			samplesType := getOutputType(parents["video"][0])
			if samplesType == "" {
				return node.Outputs
			}
			outputs := []skyhook.ExecOutput{{
				Name: "samples",
				DataType: samplesType,
			}}

			for i, parent := range parents["others"] {
				outputType := getOutputType(parent)
				if outputType == "" {
					return node.Outputs
				}
				outputs = append(outputs, skyhook.ExecOutput{
					Name: fmt.Sprintf("others%d", i),
					DataType: outputType,
				})
			}
			return outputs
		},
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
