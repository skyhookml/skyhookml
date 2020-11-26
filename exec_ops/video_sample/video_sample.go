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
	// map from keys to list of (start, end) segments
	Samples map[string][][2]int
}

func (e *VideoSample) Apply(key string, inputs []skyhook.Item) (map[string][]skyhook.Data, error) {
	samples := e.Samples[key]
	if len(samples) == 0 {
		return nil, nil
	}

	log.Printf("[video_sample %s] extracting %d samples from %s", e.Node.Name, len(samples), key)

	if len(inputs) != 1 {
		return nil, fmt.Errorf("video_sample expects exactly one input")
	}
	data, err := inputs[0].LoadData()
	if err != nil {
		return nil, err
	} else if data.Type() != skyhook.VideoType {
		return nil, fmt.Errorf("video_sample expects video input")
	}
	vdata := data.(skyhook.VideoData)

	// create map of where samples start
	startToEnd := make(map[int][]int)
	for _, sample := range samples {
		startToEnd[sample[0]] = append(startToEnd[sample[0]], sample[1])
	}

	outputs := make(map[string][]skyhook.Data)

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
	err = vdata.Iterator().Iterate(32, func(im skyhook.Image) {
		// add segments that start at this frame to the processing set
		for _, end := range startToEnd[counter] {
			sampleKey := fmt.Sprintf("%s_%d_%d", key, counter, end)
			if _, ok := processing[sampleKey]; ok {
				// duplicate interval
				continue
			}

			// for images, we can do it quickly
			if end - counter == 1 {
				outputs[sampleKey] = []skyhook.Data{skyhook.ImageData{Images: []skyhook.Image{im}}}
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
			outputs[resp.Key] = []skyhook.Data{resp.Data}
		} else if err == nil {
			err = resp.Error
		}
	}

	if err != nil {
		return nil, err
	}

	return outputs, nil
}

func (e *VideoSample) Close() {}

func init() {
	skyhook.ExecOpImpls["video_sample"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode) (skyhook.ExecOp, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured", err)
			}

			// get items in the video parent
			datasets, err := exec_ops.GetParentDatasets(url, node)
			if err != nil {
				return nil, fmt.Errorf("error getting parent datasets: %v", err)
			}
			dataset := datasets[0]
			var rawItems []skyhook.Item
			err = skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), &rawItems)
			if err != nil {
				return nil, fmt.Errorf("error getting parent dataset items: %v", err)
			}

			// only keep items that have length set, and at least params.Length
			type Item struct {
				Item skyhook.Item
				Metadata skyhook.VideoMetadata
				NumFrames int
			}
			var items []Item
			for _, item := range rawItems {
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
				items = append(items, Item{item, metadata, numFrames})
			}

			// select the samples
			samples := make(map[string][][2]int)
			if params.Mode == "random" {
				// sample item based on how many possible segments there are in the item
				// (which depends on item and segment lengths)
				weights := make([]int, len(items))
				var sum int
				for i, item := range items {
					weight := item.NumFrames - params.Length
					weights[i] = weight
					sum += weight
				}
				sampleItem := func() Item {
					r := rand.Intn(sum)
					for i, w := range weights {
						r -= w
						if r < 0 {
							return items[i]
						}
					}
					return items[len(items)-1]
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

			return &VideoSample{
				URL: url,
				Node: node,
				Params: params,
				Samples: samples,
			}, nil
		},
	}
}
