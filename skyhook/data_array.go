package skyhook

import (
	"fmt"
	"io"
)

type ArrayMetadata struct {
	Width int `json:",omitempty"`
	Height int `json:",omitempty"`
	Channels int `json:",omitempty"`

	// uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64
	Type string `json:",omitempty"`
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

func (m ArrayMetadata) BytesPerElement() int {
	return m.Width*m.Height*m.Channels*m.Size()
}

func (m ArrayMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(ArrayMetadata)
	if other_.Width > 0 {
		m.Width = other_.Width
	}
	if other_.Height > 0 {
		m.Height = other_.Height
	}
	if other_.Channels > 0 {
		m.Channels = other_.Channels
	}
	if other_.Type != "" {
		m.Type = other_.Type
	}
	return m
}

type ArrayDataSpec struct{}

func (s ArrayDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return ArrayMetadata{}
	}
	var m ArrayMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

type ArrayStreamHeader struct {
	Length int
	BytesPerElement int
}

func (s ArrayDataSpec) ReadStream(r io.Reader) (interface{}, error) {
	var header ArrayStreamHeader
	if err := ReadJsonData(r, &header); err != nil {
		return nil, err
	}
	byteList := make([][]byte, header.Length)
	for i := range byteList {
		buf := make([]byte, header.BytesPerElement)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		byteList[i] = buf
	}
	return byteList, nil
}

func (s ArrayDataSpec) WriteStream(data interface{}, w io.Writer) error {
	byteList := data.([][]byte)
	header := ArrayStreamHeader{
		Length: len(byteList),
		BytesPerElement: len(byteList[0]),
	}
	if err := WriteJsonData(header, w); err != nil {
		return err
	}
	for _, bytes := range byteList {
		if _, err := w.Write(bytes); err != nil {
			return err
		}
	}
	return nil
}

func (s ArrayDataSpec) Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	metadata_ := metadata.(ArrayMetadata)
	if format == "bin" {
		var byteList [][]byte
		for {
			buf := make([]byte, metadata_.BytesPerElement())
			_, err := io.ReadFull(r, buf)
			if err != nil {
				return nil, err
			}
			byteList = append(byteList, buf)
		}
		return byteList, nil
	}
	return nil, fmt.Errorf("unknown format %s", format)
}

func (s ArrayDataSpec) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	if format == "bin" {
		byteList := data.([][]byte)
		for _, bytes := range byteList {
			if _, err := w.Write(bytes); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("unknown format %s", format)
}

func (s ArrayDataSpec) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "bin", "bin"
}

type ArrayReader struct {
	metadata ArrayMetadata
	r io.Reader
}

func (r ArrayReader) Read(n int) (interface{}, error) {
	var byteList [][]byte
	for i := 0; i < n || n == -1; i++ {
		buf := make([]byte, r.metadata.BytesPerElement())
		_, err := io.ReadFull(r.r, buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		byteList = append(byteList, buf)
	}
	return byteList, nil
}
func (r ArrayReader) Close() {}

func (s ArrayDataSpec) Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader {
	return ArrayReader{
		metadata: metadata.(ArrayMetadata),
		r: r,
	}
}

type ArrayWriter struct {
	w io.Writer
}

func (w ArrayWriter) Write(data interface{}) error {
	byteList := data.([][]byte)
	for _, bytes := range byteList {
		_, err := w.w.Write(bytes)
		if err != nil {
			return err
		}
	}
	return nil
}
func (w ArrayWriter) Close() error { return nil }

func (s ArrayDataSpec) Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter {
	return ArrayWriter{w}
}

func (s ArrayDataSpec) Length(data interface{}) int {
	return len(data.([][]byte))
}
func (s ArrayDataSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([][]byte), more.([][]byte)...)
}
func (s ArrayDataSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([][]byte)[i:j]
}

func init() {
	DataSpecs[ArrayType] = ArrayDataSpec{}
}
