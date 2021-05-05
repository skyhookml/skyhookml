package skyhook

import (
	"encoding/json"
	"fmt"
	"io"
)

// Implement DataSpec using simple JSON-only format.
type SequenceJsonSpec interface {
	DecodeMetadata(rawMetadata string) DataMetadata
	Decode(dec *json.Decoder, n int) (data interface{}, err error)
	Encode(enc *json.Encoder, data interface{}) error
	GetEmptyData() (data interface{})
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
	data = s.Spec.GetEmptyData()
	if err := ReadJsonData(r, &data); err != nil {
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
	decoder := json.NewDecoder(r)
	return s.Spec.Decode(decoder, -1)
}

func (s SequenceJsonDataImpl) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	if format == "" {
		format = "json"
	}
	if format != "json" {
		return fmt.Errorf("format must be json")
	}
	encoder := json.NewEncoder(w)
	return s.Spec.Encode(encoder, data)
}

func (s SequenceJsonDataImpl) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "json", "json"
}

func (s SequenceJsonDataImpl) GetMetadataFromFile(fname string) (format string, metadata DataMetadata, err error) {
	metadata = s.Spec.GetEmptyMetadata()
	return "json", metadata, nil
}

type SequenceJsonReader struct {
	Spec SequenceJsonSpec
	Decoder *json.Decoder
}

func (r SequenceJsonReader) Read(n int) (interface{}, error) {
	return r.Spec.Decode(r.Decoder, n)
}

func (r SequenceJsonReader) Close() {}

func (s SequenceJsonDataImpl) Reader(format string, metadata DataMetadata, r io.Reader) SequenceReader {
	decoder := json.NewDecoder(r)
	return SequenceJsonReader{s.Spec, decoder}
}

type SequenceJsonWriter struct {
	Spec SequenceJsonSpec
	Encoder *json.Encoder
}

func (w SequenceJsonWriter) Write(data interface{}) error {
	return w.Spec.Encode(w.Encoder, data)
}

func (w SequenceJsonWriter) Close() error { return nil }

func (s SequenceJsonDataImpl) Writer(format string, metadata DataMetadata, w io.Writer) SequenceWriter {
	encoder := json.NewEncoder(w)
	return SequenceJsonWriter{s.Spec, encoder}
}

func (s SequenceJsonDataImpl) Length(data interface{}) int { return s.Spec.Length(data) }
func (s SequenceJsonDataImpl) Append(data interface{}, more interface{}) interface{} { return s.Spec.Append(data, more) }
func (s SequenceJsonDataImpl) Slice(data interface{}, i int, j int) interface{} { return s.Spec.Slice(data, i, j) }
