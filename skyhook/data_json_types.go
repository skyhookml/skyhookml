package skyhook

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

// Implement DataSpec using simple JSON-only format.
type SequenceJsonSpec interface {
	DecodeMetadata(rawMetadata string) DataMetadata
	DecodeData(bytes []byte) (interface{}, error)
	GetEmptyMetadata() (metadata DataMetadata)

	Length(data interface{}) int
	Append(data interface{}, more interface{}) interface{}
	Slice(data interface{}, i int, j int) interface{}
}

type SequenceJsonDataImpl struct {
	Spec SequenceJsonSpec
}

func (s SequenceJsonDataImpl) DecodeMetadata(rawMetadata string) DataMetadata {
	return s.Spec.DecodeMetadata(rawMetadata)
}

func (s SequenceJsonDataImpl) ReadStream(r io.Reader) (data interface{}, err error) {
	// Copied from ReadJsonData.
	// But instead of decoding directly, we pass to spec.DecodeData.
	blen := make([]byte, 4)
	if _, err := io.ReadFull(r, blen); err != nil {
		return nil, err
	}
	bytes := make([]byte, binary.BigEndian.Uint32(blen))
	if _, err := io.ReadFull(r, bytes); err != nil {
		return nil, err
	}
	data, err = s.Spec.DecodeData(bytes)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s SequenceJsonDataImpl) WriteStream(data interface{}, w io.Writer) error {
	if err := WriteJsonData(data, w); err != nil {
		return err
	}
	return nil
}

func (s SequenceJsonDataImpl) Read(format string, metadata DataMetadata, r io.Reader) (data interface{}, err error) {
	if format != "json" {
		return nil, fmt.Errorf("format must be json")
	}
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	data, err = s.Spec.DecodeData(bytes)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s SequenceJsonDataImpl) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	if format == "" {
		format = "json"
	}
	if format != "json" {
		return fmt.Errorf("format must be json")
	}
	bytes := JsonMarshal(data)
	_, err := w.Write(bytes)
	return err
}

func (s SequenceJsonDataImpl) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "json", "json"
}

func (s SequenceJsonDataImpl) GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error) {
	metadata = s.Spec.GetEmptyMetadata()
	return "json", metadata, nil
}

func (s SequenceJsonDataImpl) Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader {
	data, err := s.Read(format, metadata, r)
	if err != nil {
		return ErrorSequenceReader{err}
	}
	return &SliceReader{
		Data: data,
		Spec: s,
	}
}

func (s SequenceJsonDataImpl) Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter {
	return &SliceWriter{
		Spec: s,
		Format: format,
		Metadata: metadata,
		Writer: w,
	}
}

func (s SequenceJsonDataImpl) Length(data interface{}) int { return s.Spec.Length(data) }
func (s SequenceJsonDataImpl) Append(data interface{}, more interface{}) interface{} { return s.Spec.Append(data, more) }
func (s SequenceJsonDataImpl) Slice(data interface{}, i int, j int) interface{} { return s.Spec.Slice(data, i, j) }
