package skyhook

import (
	"fmt"
	"io"
	"path/filepath"
)


type ImageDataSpec struct{}

func (s ImageDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

type ImageStreamHeader struct {
	Width int
	Height int
	Channels int
	Length int
	BytesPerElement int
}

// Image is usually stored as Image but may become []Image since we support
// slice operations (so that operations can process Video/Image in the same way
// through SynchronizedReader).
// So this helper function tries both and returns just the Image.
func (s ImageDataSpec) getImage(data interface{}) Image {
	if image, ok := data.(Image); ok {
		return image
	}
	return data.([]Image)[0]
}

func (s ImageDataSpec) ReadStream(r io.Reader) (interface{}, error) {
	var header ImageStreamHeader
	if err := ReadJsonData(r, &header); err != nil {
		return nil, err
	}
	bytes := make([]byte, header.Width*header.Height*3)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return nil, err
	}
	image := Image{
		Width: header.Width,
		Height: header.Height,
		Bytes: bytes,
	}
	return image, nil
}

func (s ImageDataSpec) WriteStream(data interface{}, w io.Writer) error {
	image := s.getImage(data)
	header := ImageStreamHeader{
		Width: image.Width,
		Height: image.Height,
		Channels: 3,
		Length: 1,
		BytesPerElement: len(image.Bytes),
	}
	if err := WriteJsonData(header, w); err != nil {
		return err
	}
	if _, err := w.Write(image.Bytes); err != nil {
		return err
	}
	return nil
}

func (s ImageDataSpec) Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	var image Image
	if format == "jpeg" {
		image, err = ImageFromJPGReader(r)
	} else if format == "png" {
		image, err = ImageFromPNGReader(r)
	} else {
		err = fmt.Errorf("unknown format %s", format)
	}
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (s ImageDataSpec) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	image := s.getImage(data)
	if format == "jpeg" {
		bytes, err := image.AsJPG()
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		return err
	} else if format == "png" {
		bytes, err := image.AsPNG()
		if err != nil {
			return err
		}
		_, err = w.Write(bytes)
		return err
	}
	return fmt.Errorf("unknown format %s", format)
}

func (s ImageDataSpec) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "jpg", "jpeg"
}

func (s ImageDataSpec) Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader {
	return NewSliceReader(s, format, metadata, r)
}

func (s ImageDataSpec) Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter {
	return &SliceWriter{
		Spec: s,
		Format: format,
		Metadata: metadata,
		Writer: w,
	}
}

func (s ImageDataSpec) Length(data interface{}) int {
	return 1
}
func (s ImageDataSpec) Append(data interface{}, more interface{}) interface{} {
	panic(fmt.Errorf("ImageDataSpec.Append not supported"))
}
func (s ImageDataSpec) Slice(data interface{}, i int, j int) interface{} {
	if i == 0 && j == 1 {
		image := s.getImage(data)
		return []Image{image}
	}
	panic(fmt.Errorf("ImageDataSpec.Slice not supported"))
}

func (s ImageDataSpec) GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error) {
	ext := filepath.Ext(fname)
	if ext == ".jpg" || ext == ".jpeg" {
		return "jpeg", NoMetadata{}, nil
	} else if ext == ".png" {
		return "png", NoMetadata{}, nil
	}
	return "", nil, fmt.Errorf("unrecognized image extension %s in [%s]", ext, fname)
}

func (s ImageDataSpec) GetExtFromFormat(format string) (ext string) {
	if format == "jpeg" {
		return "jpg"
	} else if format == "png" {
		return "png"
	}
	return ""
}

func init() {
	DataSpecs[ImageType] = ImageDataSpec{}
}
