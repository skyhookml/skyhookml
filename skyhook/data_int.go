package skyhook

import (
	"io"
	"io/ioutil"
)

type IntData struct {
	Ints []int
}

func (d IntData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d.Ints, w)
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
	return nil
}

// SliceData
func (d IntData) Length() int {
	return len(d.Ints)
}
func (d IntData) Slice(i, j int) Data {
	return IntData{Ints: d.Ints[i:j]}
}
func (d IntData) Append(other Data) Data {
	return IntData{
		Ints: append(d.Ints, other.(IntData).Ints...),
	}
}

func (d IntData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[IntType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data IntData
			if err := ReadJsonData(r, &data.Ints); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var data IntData
			ReadJSONFile(fname, &data.Ints)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			var data IntData
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
