package render

import (
	"../../skyhook"
	"../../exec_ops"

	"bytes"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
)

var Colors = [][3]uint8{
	[3]uint8{255, 0, 0},
	[3]uint8{0, 255, 0},
	[3]uint8{0, 0, 255},
	[3]uint8{255, 255, 0},
	[3]uint8{0, 255, 255},
	[3]uint8{255, 0, 255},
	[3]uint8{0, 51, 51},
	[3]uint8{51, 153, 153},
	[3]uint8{102, 0, 51},
	[3]uint8{102, 51, 204},
	[3]uint8{102, 153, 204},
	[3]uint8{102, 255, 204},
	[3]uint8{153, 102, 102},
	[3]uint8{204, 102, 51},
	[3]uint8{204, 255, 102},
	[3]uint8{255, 255, 204},
	[3]uint8{121, 125, 127},
	[3]uint8{69, 179, 157},
	[3]uint8{250, 215, 160},
}


type Render struct {
	URL string
	Node skyhook.ExecNode
	Dataset skyhook.Dataset
}

func (e *Render) Parallelism() int {
	return runtime.NumCPU()
}

func renderFrame(datas []skyhook.Data) (skyhook.Image, error) {
	var canvas skyhook.Image
	var canvases []skyhook.Image
	for _, data := range datas {
		if data.Type() == skyhook.ImageType {
			canvas = data.(skyhook.ImageData).Images[0].Copy()
			canvases = append(canvases, canvas)
		}

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
			detectionData := data.(skyhook.DetectionData)
			detections := detectionData.Detections[0]
			detectionDims := detectionData.Metadata.CanvasDims
			targetDims := [2]int{canvas.Width, canvas.Height}
			for _, d := range detections {
				if detectionDims[0] != 0 && detectionDims != targetDims {
					d = d.Rescale(detectionDims, targetDims)
				}
				color := Colors[d.TrackID % len(Colors)]
				canvas.DrawRectangle(d.Left, d.Top, d.Right, d.Bottom, 2, color)
			}
		}
	}

	if len(canvases) > 1 {
		// stack the canvases vertically
		var dims [2]int
		for _, im := range canvases {
			if im.Width > dims[0] {
				dims[0] = im.Width
			}
			dims[1] += im.Height
		}
		canvas = skyhook.NewImage(dims[0], dims[1])
		heightOffset := 0
		for _, im := range canvases {
			canvas.DrawImage(0, heightOffset, im)
			heightOffset += im.Height
		}
	}

	return canvas, nil
}

func (e *Render) Apply(task skyhook.ExecTask) error {
	inputDatas := make([]skyhook.Data, len(task.Items["inputs"]))
	for i, input := range task.Items["inputs"] {
		data, err := input[0].LoadData()
		if err != nil {
			return err
		}
		inputDatas[i] = data
	}

	// first input should be video data or image data
	// there may be multiple video/image that we want to render
	// but they should all be the same type (and, if video, they must have same framerates)
	// the output will have all the video/image stacked vertically
	if inputDatas[0].Type() == skyhook.VideoType {
		// use video metadata to determine the canvas dimensions
		var dims [2]int
		var videoMetadata skyhook.VideoMetadata
		for _, data := range inputDatas {
			if data.Type() != skyhook.VideoType {
				continue
			}
			videoData := data.(skyhook.VideoData)
			videoMetadata = videoData.Metadata
			curDims := videoData.Metadata.Dims
			if curDims[0] > dims[0] {
				dims[0] = curDims[0]
			}
			dims[1] += curDims[1]
		}

		imCh := make(chan skyhook.Image)
		doneCh := make(chan error)
		rd, cmd := skyhook.MakeVideo(&skyhook.ChanReader{imCh}, dims, videoMetadata.Framerate)
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
			return perFrameErr
		}

		output := skyhook.VideoData{
			Bytes: buf.Bytes(),
			Metadata: videoMetadata,
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
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
			op := &Render{url, node, outputDatasets["output"]}
			return op, nil
		},
		GetOutputs: func(url string, node skyhook.ExecNode) []skyhook.ExecOutput {
			// whether we output video or image depends on the first input
			parents := node.GetParents()
			if len(parents["inputs"]) == 0 {
				return node.Outputs
			}
			inputType, err := exec_ops.ParentToDataType(url, parents["inputs"][0])
			if err != nil {
				log.Printf("[render] warning: unable to compute outputs: %v", err)
				return node.Outputs
			}
			return []skyhook.ExecOutput{{
				Name: "output",
				DataType: inputType,
			}}
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
