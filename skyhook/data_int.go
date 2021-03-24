package skyhook

import (
	"io"
	"io/ioutil"
)

type IntMetadata struct {
	Categories []string `json:",omitempty"`
}

type IntData struct {
	Ints []int
	Metadata IntMetadata
}

func (d IntData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d, w)
}

func (d IntData) Encode(format string, w io.Writer) error {
	_, err := w.Write(JsonMarshal(d.Ints))
	return err
}

func (d IntData) Type() DataType {
	return IntType
}

func (d IntData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d IntData) GetMetadata() interface{} {
	return d.Metadata
}

// SliceData
func (d IntData) Length() int {
	return len(d.Ints)
}
func (d IntData) Slice(i, j int) Data {
	return IntData{
		Ints: d.Ints[i:j],
		Metadata: d.Metadata,
	}
}
func (d IntData) Append(other_ Data) Data {
	other := other_.(IntData)
	return IntData{
		Ints: append(d.Ints, other.Ints...),
		Metadata: other.Metadata,
	}
}

func (d IntData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[IntType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data IntData
			if err := ReadJsonData(r, &data); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var metadata IntMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			data := IntData{Metadata: metadata}
			ReadJSONFile(fname, &data.Ints)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata IntMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			data := IntData{Metadata: metadata}
			JsonUnmarshal(bytes, &data.Ints)
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "json", "", nil
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: IntData{}}
		},
		ChunkType: IntType,
	}
}
