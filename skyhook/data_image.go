package skyhook

import (
	"fmt"
	"io"
	"path/filepath"
)

type ImageData struct {
	Images []Image
}

type ImageStreamHeader struct {
	Length int
	Width int
	Height int
}

func (d ImageData) EncodeStream(w io.Writer) error {
	WriteJsonData(ImageStreamHeader{
		Length: len(d.Images),
		Width: d.Images[0].Width,
		Height: d.Images[0].Height,
	}, w)
	for _, image := range d.Images {
		if _, err := w.Write(image.Bytes); err != nil {
			return err
		}
	}
	return nil
}

func (d ImageData) Encode(format string, w io.Writer) error {
	if len(d.Images) != 1 {
		return fmt.Errorf("image data can only be encoded with one image")
	}
	image := d.Images[0]
	if format == "jpeg" {
		_, err := w.Write(image.AsJPG())
		return err
	} else if format == "png" {
		_, err := w.Write(image.AsPNG())
		return err
	}
	return fmt.Errorf("unknown format %s", format)
}

func (d ImageData) Type() DataType {
	return ImageType
}

func (d ImageData) GetDefaultExtAndFormat() (string, string) {
	return "jpg", "jpeg"
}

func (d ImageData) GetMetadata() interface{} {
	return nil
}

// SliceData
func (d ImageData) Length() int {
	return len(d.Images)
}
func (d ImageData) Slice(i, j int) Data {
	return ImageData{Images: d.Images[i:j]}
}
func (d ImageData) Append(other Data) Data {
	return ImageData{
		Images: append(d.Images, other.(ImageData).Images...),
	}
}

func (d ImageData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[ImageType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var header ImageStreamHeader
			if err := ReadJsonData(r, &header); err != nil {
				return nil, err
			}
			var images []Image
			for i := 0; i < header.Length; i++ {
				bytes := make([]byte, header.Width*header.Height*3)
				if _, err := io.ReadFull(r, bytes); err != nil {
					return nil, err
				}
				images = append(images, Image{
					Width: header.Width,
					Height: header.Height,
					Bytes: bytes,
				})
			}
			return ImageData{Images: images}, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var image Image
			if format == "jpeg" {
				image = ImageFromJPGReader(r)
			} else if format == "png" {
				image = ImageFromPNGReader(r)
			} else {
				return nil, fmt.Errorf("unknown format %s", format)
			}
			return ImageData{
				Images: []Image{image},
			}, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			ext := filepath.Ext(fname)
			if ext == ".jpg" || ext == ".jpeg" {
				return "jpeg", "", nil
			} else if ext == ".png" {
				return "png", "", nil
			}
			return "", "", fmt.Errorf("unrecognized image extension %s", ext)
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: ImageData{}}
		},
		ChunkType: ImageType,
	}
}
