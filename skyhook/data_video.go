package skyhook

import (
	"fmt"
	"io"
)

type VideoMetadata struct {
	// view settings that can be adjusted
	Dims [2]int `json:",omitempty"`
	Framerate [2]int `json:",omitempty"`

	// cached properties that don't make sense to adjust
	Duration float64 `json:",omitempty"`
}

// Approximate number of frames in this video.
func (m VideoMetadata) NumFrames() int {
	return int(m.Duration * float64(m.Framerate[0])) / m.Framerate[1]
}

func (m VideoMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(VideoMetadata)
	if other_.Dims[0] > 0 {
		m.Dims = other_.Dims
	}
	if other_.Framerate[0] > 0 {
		m.Framerate = other_.Framerate
	}
	if other_.Duration > 0 {
		m.Duration = other_.Duration
	}
	return m
}

type VideoDataSpec struct{}

func (s VideoDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return VideoMetadata{}
	}
	var m VideoMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

type VideoStreamHeader struct {
	Width int
	Height int
	Channels int
	Length int
	BytesPerElement int
}

func (s VideoDataSpec) ReadStream(r io.Reader) (data interface{}, err error) {
	var header VideoStreamHeader
	if err := ReadJsonData(r, &header); err != nil {
		return nil, err
	}
	images := make([]Image, header.Length)
	for i := range images {
		bytes := make([]byte, header.Width*header.Height*3)
		if _, err := io.ReadFull(r, bytes); err != nil {
			return nil, err
		}
		images[i] = Image{
			Width: header.Width,
			Height: header.Height,
			Bytes: bytes,
		}
	}
	return images, nil
}

func (s VideoDataSpec) WriteStream(data interface{}, w io.Writer) error {
	images := data.([]Image)
	header := VideoStreamHeader{
		Width: images[0].Width,
		Height: images[0].Height,
		Channels: 3,
		Length: len(images),
		BytesPerElement: len(images[0].Bytes),
	}
	if err := WriteJsonData(header, w); err != nil {
		return err
	}
	for _, image := range images {
		if _, err := w.Write(image.Bytes); err != nil {
			return err
		}
	}
	return nil
}

func (s VideoDataSpec) Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	return nil, fmt.Errorf("unsupported")
}
func (s VideoDataSpec) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	return fmt.Errorf("unsupported")
}

func (s VideoDataSpec) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "mp4", "mp4"
}

// VideoIterator duals as a SequenceReader for video.
type VideoIterator struct {
	Metadata VideoMetadata
	Fname string
	Reader io.Reader

	rd *FfmpegReader
	err error

	start int
	length int
}

func (it *VideoIterator) init() {
	it.rd = ReadFfmpeg(it.Fname, it.Metadata.Dims, it.Metadata.Framerate, ReadFfmpegOptions{
		Fname: it.Fname,
		Reader: it.Reader,
		Start: it.start,
		Length: it.length,
	})
}

func (it *VideoIterator) Get(n int) ([]Image, error) {
	if it.rd == nil {
		it.init()
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

func (it *VideoIterator) Read(n int) (interface{}, error) {
	images, err := it.Get(n)
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (it *VideoIterator) Close() {
	it.rd.Close()
}

type VideoBuilder struct {
	Metadata VideoMetadata
	Fname string
	Writer io.Writer

	ch chan Image

	cmd *Cmd
	dims [2]int
	rate [2]int
	count int

	// if donech receives nil, err should be set by MakeVideo reader
	err error
}

func (b *VideoBuilder) Write(chunk interface{}) error {
	images := chunk.([]Image)
	if len(images) == 0 {
		return nil
	}

	if b.ch == nil {
		// need to start the MakeVideo goroutine
		if b.Metadata.Dims[0] != 0 {
			b.dims = b.Metadata.Dims
		} else {
			exampleIm := images[0]
			b.dims = [2]int{exampleIm.Width, exampleIm.Height}
		}
		if b.Metadata.Framerate[0] != 0 {
			b.rate = b.Metadata.Framerate
		} else {
			b.rate = [2]int{10, 1}
		}

		b.ch = make(chan Image)

		b.cmd = MakeVideo(&ChanReader{b.ch}, b.dims, b.rate, MakeVideoOptions{
			Fname: b.Fname,
			Writer: b.Writer,
		})
	}

	for _, image := range images {
		b.ch <- image
	}
	b.count += len(images)
	return nil
}

func (b *VideoBuilder) Close() (error) {
	close(b.ch)
	err := b.cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Get the duration of the written video.
// Must only be called after Close().
func (b *VideoBuilder) GetDuration() float64 {
	return float64(b.count*b.rate[1])/float64(b.rate[0])
}

func (s VideoDataSpec) Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader {
	return &VideoIterator{
		Metadata: metadata.(VideoMetadata),
		Reader: r,
	}
}

func (s VideoDataSpec) Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter {
	return &VideoBuilder{
		Metadata: metadata.(VideoMetadata),
		Writer: w,
	}
}

func (s VideoDataSpec) Length(data interface{}) int {
	return len(data.([]Image))
}
func (s VideoDataSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([]Image), more.([]Image)...)
}
func (s VideoDataSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([]Image)[i:j]
}

func (s VideoDataSpec) FileReader(format string, metadata DataMetadata, fname string) SequenceReader {
	return &VideoIterator{
		Metadata: metadata.(VideoMetadata),
		Fname: fname,
	}
}

func (s VideoDataSpec) FileWriter(format string, metadata DataMetadata, fname string) SequenceWriter {
	return &VideoBuilder{
		Metadata: metadata.(VideoMetadata),
		Fname: fname,
	}
}

func (s VideoDataSpec) ReadSlice(format string, metadata DataMetadata, fname string, i, j int) SequenceReader {
	return &VideoIterator{
		Metadata: metadata.(VideoMetadata),
		Fname: fname,
		start: i,
		length: j-i,
	}
}

// Use ffprobe to get the resolution and duration of the video.
// Framerate currently defaults to 10 fps.
func (s VideoDataSpec) GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error) {
	width, height, duration, probeErr := Ffprobe(fname)
	if probeErr != nil {
		return "", nil, probeErr
	}
	metadata = VideoMetadata{
		Dims: [2]int{width, height},
		Framerate: [2]int{10, 1},
		Duration: duration,
	}
	return "mp4", metadata, nil
}

func init() {
	DataSpecs[VideoType] = VideoDataSpec{}
}
