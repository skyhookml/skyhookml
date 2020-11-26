package skyhook

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type VideoMetadata struct {
	// view settings that can be adjusted
	Dims [2]int
	Framerate [2]int

	// cached properties that don't make sense to adjust
	Duration float64
}

// Approximate number of frames in this video.
func (m VideoMetadata) NumFrames() int {
	return int(m.Duration * float64(m.Framerate[0])) / m.Framerate[1]
}

type VideoData struct {
	// Video source is either a filename or in-memory bytes.
	Fname string
	Bytes []byte

	Metadata VideoMetadata
}

func (d VideoData) writeBytes(w io.Writer) error {
	if d.Bytes != nil {
		if _, err := w.Write(d.Bytes); err != nil {
			return err
		}
	} else {
		file, err := os.Open(d.Fname)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(w, file); err != nil {
			return err
		}
	}

	return nil
}

func (d VideoData) EncodeStream(w io.Writer) error {
	// stream encoding consists of metadata followed by video bytes
	// TODO: should send the format of the video too
	metaBytes := JsonMarshal(d.Metadata)
	hlen := make([]byte, 4)
	binary.BigEndian.PutUint32(hlen, uint32(len(metaBytes)))
	w.Write(hlen)
	if _, err := w.Write(metaBytes); err != nil {
		return err
	}

	if err := d.writeBytes(w); err != nil {
		return err
	}

	return nil
}

func (d VideoData) Encode(format string, w io.Writer) error {
	// currently we do not support format, so we just write the bytes to the writer
	return d.writeBytes(w)
}

func (d VideoData) Type() DataType {
	return VideoType
}

func (d VideoData) GetDefaultExtAndFormat() (string, string) {
	return "mp4", "mp4"
}

func (d VideoData) GetMetadata() interface{} {
	return d.Metadata
}

func (d VideoData) Iterator() *VideoIterator {
	return &VideoIterator{Data: d}
}

func (d VideoData) Reader() DataReader {
	return d.Iterator()
}

// VideoIterator duals as a DataReader for video (producing ImageData chunks)
type VideoIterator struct {
	Data VideoData
	rd *FfmpegReader
	err error
}

func (it *VideoIterator) start() {
	dims := it.Data.Metadata.Dims
	rate := it.Data.Metadata.Framerate

	if it.Data.Bytes != nil {
		cmd := Command(
			"ffmpeg-iter", CommandOptions{OnlyDebug: true},
			"ffmpeg",
			"-threads", "2",
			"-f", "mp4", "-i", "-",
			"-c:v", "rawvideo", "-pix_fmt", "rgb24", "-f", "rawvideo",
			"-vf", fmt.Sprintf("scale=%dx%d,fps=fps=%d/%d", dims[0], dims[1], rate[0], rate[1]),
			"-",
		)

		go func() {
			stdin := cmd.Stdin()
			stdin.Write(it.Data.Bytes)
			stdin.Close()
		}()

		it.rd = &FfmpegReader{
			Cmd: cmd,
			Stdout: cmd.Stdout(),
			Width: dims[0],
			Height: dims[1],
			Buf: make([]byte, dims[0]*dims[1]*3),
		}
	} else {
		it.rd = ReadFfmpeg(it.Data.Fname, dims, rate)
	}
}

func (it *VideoIterator) Get(n int) ([]Image, error) {
	if it.rd == nil {
		it.start()
	}

	if it.err != nil {
		return nil, it.err
	}

	var ims []Image

	for len(ims) == 0 || (n > 0 && len(ims) < n) {
		im, err := it.rd.Read()
		if err == io.EOF {
			it.err = err
			break
		} else if err != nil {
			it.err = err
			return nil, it.err
		}
		ims = append(ims, im)
	}

	return ims, nil
}

func (it *VideoIterator) Iterate(n int, f func(Image)) error {
	defer it.Close()
	for {
		ims, err := it.Get(n)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		for _, im := range ims {
			f(im)
		}
	}
}

func (it *VideoIterator) Read(n int) (Data, error) {
	images, err := it.Get(n)
	if err != nil {
		return nil, err
	}
	return ImageData{Images: images}, nil
}

func (it *VideoIterator) Close() {
	it.rd.Close()
}

type VideoBuilder struct {
	ch chan Image

	// once ch is closed, MakeVideo reader will pass produced bytes to donech
	// or nil, if there was an error
	donech chan []byte

	dims [2]int
	rate [2]int
	count int

	// if donech receives nil, err should be set by MakeVideo reader
	err error
}

func (b *VideoBuilder) Write(chunk Data) error {
	imageData := chunk.(ImageData)
	if len(imageData.Images) == 0 {
		return nil
	}

	if b.ch == nil {
		// need to start the MakeVideo goroutine
		// this is done here since we need to know image dimensions first
		// in the future maybe dimensions and framerate should be passed as metadata
		exampleIm := imageData.Images[0]
		b.dims = [2]int{exampleIm.Width, exampleIm.Height}
		b.rate = [2]int{10, 1} // TODO

		b.ch = make(chan Image)
		b.donech = make(chan []byte)

		r, cmd := MakeVideo(&ChanReader{b.ch}, b.dims, b.rate)
		go func() {
			buf := new(bytes.Buffer)
			io.Copy(buf, r)
			err := cmd.Wait()
			if err != nil {
				b.err = err
				b.donech <- nil
				return
			}
			b.donech <- buf.Bytes()
		}()
	}

	for _, image := range imageData.Images {
		b.ch <- image
	}
	b.count += len(imageData.Images)
	return nil
}

func (b *VideoBuilder) Close() (Data, error) {
	close(b.ch)
	bytes := <- b.donech
	if b.err != nil {
		return nil, b.err
	}
	return VideoData{
		Bytes: bytes,
		Metadata: VideoMetadata{
			Dims: b.dims,
			Framerate: b.rate,
			Duration: float64(b.count*b.rate[1])/float64(b.rate[0]),
		},
	}, nil
}

func init() {
	DataImpls[VideoType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			hlen := make([]byte, 4)
			if _, err := io.ReadFull(r, hlen); err != nil {
				return nil, err
			}
			l := int(binary.BigEndian.Uint32(hlen))
			metaBytes := make([]byte, l)
			if _, err := io.ReadFull(r, metaBytes); err != nil {
				return nil, err
			}
			var metadata VideoMetadata
			JsonUnmarshal(metaBytes, &metadata)

			videoBytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return VideoData{
				Bytes: videoBytes,
				Metadata: metadata,
			}, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var metadata VideoMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)
			return VideoData{
				Fname: fname,
				Metadata: metadata,
			}, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata VideoMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)
			videoBytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return VideoData{
				Bytes: videoBytes,
				Metadata: metadata,
			}, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			width, height, duration, probeErr := Ffprobe(fname)
			if probeErr != nil {
				err = probeErr
				return
			}
			metadata := VideoMetadata{
				Dims: [2]int{width, height},
				Framerate: [2]int{10, 1},
				Duration: duration,
			}
			metadataRaw = string(JsonMarshal(metadata))
			return
		},
		Builder: func() ChunkBuilder {
			return &VideoBuilder{}
		},
		ChunkType: ImageType,
	}
}
