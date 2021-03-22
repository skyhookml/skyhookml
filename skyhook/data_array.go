package skyhook

import (
	"fmt"
	"io"
	"io/ioutil"
)

type ArrayMetadata struct {
	Width int
	Height int
	Channels int

	// uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64
	Type string
}

// Bytes per primitive.
func (m ArrayMetadata) Size() int {
	switch m.Type {
	case "uint8", "int8":
		return 1
	case "uint16", "int16":
		return 2
	case "uint32", "int32", "float32":
		return 4
	case "uint64", "int64", "float64":
		return 8
	default:
		panic(fmt.Errorf("unknown array type %s", m.Type))
	}
}

func (m ArrayMetadata) BytesPerItem() int {
	return m.Width*m.Height*m.Channels*m.Size()
}

type ArrayHeader struct {
	ArrayMetadata
	Length int
}

type ArrayData struct {
	Bytes []byte
	Metadata ArrayMetadata
}

func (d ArrayData) EncodeStream(w io.Writer) error {
	WriteJsonData(ArrayHeader{
		ArrayMetadata: d.Metadata,
		Length: d.Length(),
	}, w)
	if _, err := w.Write(d.Bytes); err != nil {
		return err
	}
	return nil
}

func (d ArrayData) Encode(format string, w io.Writer) error {
	if format == "bin" {
		_, err := w.Write(d.Bytes)
		return err
	}
	return fmt.Errorf("unknown format %s", format)
}

func (d ArrayData) Type() DataType {
	return ArrayType
}

func (d ArrayData) GetDefaultExtAndFormat() (string, string) {
	return "bin", "bin"
}

func (d ArrayData) GetMetadata() interface{} {
	return d.Metadata
}

// SliceData
func (d ArrayData) Length() int {
	return len(d.Bytes) / d.Metadata.BytesPerItem()
}
func (d ArrayData) Slice(i, j int) Data {
	perItem := d.Metadata.BytesPerItem()
	return ArrayData{
		Metadata: d.Metadata,
		Bytes: d.Bytes[i*perItem:j*perItem],
	}
}
func (d ArrayData) Append(other Data) Data {
	other_ := other.(ArrayData)
	return ArrayData{
		Metadata: other_.Metadata,
		Bytes: append(d.Bytes, other_.Bytes...),
	}
}

func (d ArrayData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[ArrayType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var header ArrayHeader
			if err := ReadJsonData(r, &header); err != nil {
				return nil, err
			}
			bytes := make([]byte, header.Length*header.BytesPerItem())
			if _, err := io.ReadFull(r, bytes); err != nil {
				return nil, err
			}
			return ArrayData{
				Metadata: header.ArrayMetadata,
				Bytes: bytes,
			}, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata ArrayMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			var bytes []byte
			if format == "bin" {
				var err error
				bytes, err = ioutil.ReadAll(r)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("unknown format %s", format)
			}

			return ArrayData{
				Metadata: metadata,
				Bytes: bytes,
			}, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "", "", fmt.Errorf("array metadata cannot be determined from file")
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: ArrayData{}}
		},
		ChunkType: ArrayType,
	}
}
