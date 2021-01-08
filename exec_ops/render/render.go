package render

import (
	"../../skyhook"
	"../../exec_ops"

	"bytes"
	"fmt"
	"io"
	"runtime"
	"strconv"
)

type Render struct {
	URL string
	Node skyhook.ExecNode
	Dataset skyhook.Dataset
}

func (e *Render) Parallelism() int {
	return runtime.NumCPU()
}

func renderFrame(datas []skyhook.Data) (skyhook.Image, error) {
	canvas := datas[0].(skyhook.ImageData).Images[0].Copy()
	for _, data := range datas[1:] {
		if data.Type() == skyhook.IntType {
			x := data.(skyhook.IntData).Ints[0]
			canvas.DrawText(skyhook.RichText{Text: strconv.Itoa(x)})
		} else if data.Type() == skyhook.ShapeType {
			shapes := data.(skyhook.ShapeData).Shapes[0]
			for _, shape := range shapes {
				if shape.Type == "box" {
					bounds := shape.Bounds()
					canvas.DrawRectangle(bounds[0], bounds[1], bounds[2], bounds[3], 2, [3]uint8{255, 0, 0})
				} else if shape.Type == "line" {
					canvas.DrawLine(shape.Points[0][0], shape.Points[0][1], shape.Points[1][0], shape.Points[1][1], 1, [3]uint8{255, 0, 0})
				}
			}
		} else if data.Type() == skyhook.DetectionType {
			detections := data.(skyhook.DetectionData).Detections[0]
			for _, d := range detections {
				canvas.DrawRectangle(d.Left, d.Top, d.Right, d.Bottom, 2, [3]uint8{255, 0, 0})
			}
		}
	}
	return canvas, nil
}

func (e *Render) Apply(task skyhook.ExecTask) error {
	inputDatas := make([]skyhook.Data, len(task.Items))
	for i, input := range task.Items {
		data, err := input.LoadData()
		if err != nil {
			return err
		}
		inputDatas[i] = data
	}

	// first input should be video data or image data
	if inputDatas[0].Type() == skyhook.VideoType {
		// use video data to get the canvas width/height for rendering
		videoData := inputDatas[0].(skyhook.VideoData)
		dims := videoData.Metadata.Dims

		imCh := make(chan skyhook.Image)
		doneCh := make(chan error)
		rd, cmd := skyhook.MakeVideo(&skyhook.ChanReader{imCh}, dims, videoData.Metadata.Framerate)
		// save encoded video to buffer in background
		buf := new(bytes.Buffer)
		go func() {
			_, err := io.Copy(buf, rd)
			cmd.Wait()

			// in case ffmpeg failed prematurely, make sure we finish capturing the writes
			for _ = range imCh {}

			doneCh <- err
		}()

		perFrameErr := skyhook.PerFrame(inputDatas, func(pos int, datas []skyhook.Data) error {
			im, err := renderFrame(datas)
			if err != nil {
				return err
			}
			imCh <- im
			return nil
		})
		close(imCh)

		// check donech err first, since we need to make sure we read from donech
		err := <- doneCh
		if err != nil {
			return err
		}

		if perFrameErr != nil {
			return err
		}

		output := skyhook.VideoData{
			Bytes: buf.Bytes(),
			Metadata: videoData.Metadata,
		}
		return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, output)
	} else if inputDatas[0].Type() == skyhook.ImageType {
		var output skyhook.ImageData
		err := skyhook.PerFrame(inputDatas, func(pos int, datas []skyhook.Data) error {
			im, err := renderFrame(datas)
			if err != nil {
				return err
			}
			output.Images = append(output.Images, im)
			return nil
		})
		if err != nil {
			return err
		}
		return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, output)
	} else {
		return fmt.Errorf("first input must be either video or image")
	}
}

func (e *Render) Close() {}

func init() {
	skyhook.ExecOpImpls["render"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode, items [][]skyhook.Item, outputDatasets []skyhook.Dataset) (skyhook.ExecOp, []skyhook.ExecTask, error) {
			op := &Render{url, node, outputDatasets[0]}
			tasks := exec_ops.SimpleTasks(url, node, items)
			return op, tasks, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
	}
}
