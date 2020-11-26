package render

import (
	"../../skyhook"

	"bytes"
	"io"
	"strconv"
)

type Render struct {
	URL string
	Node skyhook.ExecNode
}

func (e *Render) Apply(key string, inputs []skyhook.Item) (map[string][]skyhook.Data, error) {
	inputDatas := make([]skyhook.Data, len(inputs))
	for i, input := range inputs {
		data, err := input.LoadData()
		if err != nil {
			return nil, err
		}
		inputDatas[i] = data
	}

	// first input should be video data
	// we use this to get the canvas width/height for rendering
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
		canvas := datas[0].(skyhook.ImageData).Images[0].Copy()
		for _, data := range datas[1:] {
			if data.Type() == skyhook.IntType {
				x := data.(skyhook.IntData).Ints[0]
				canvas.DrawText(skyhook.RichText{Text: strconv.Itoa(x)})
			}
		}
		imCh <- canvas
		return nil
	})
	close(imCh)

	// check donech err first, since we need to make sure we read from donech
	err := <- doneCh
	if err != nil {
		return nil, err
	}

	if perFrameErr != nil {
		return nil, err
	}

	output := skyhook.VideoData{
		Bytes: buf.Bytes(),
		Metadata: videoData.Metadata,
	}
	return map[string][]skyhook.Data{
		key: {output},
	}, nil
}

func (e *Render) Close() {}

func init() {
	skyhook.ExecOpImpls["render"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		Prepare: func(url string, node skyhook.ExecNode) (skyhook.ExecOp, error) {
			return &Render{url, node}, nil
		},
	}
}
