package skyhook

import (
	"io"
	"io/ioutil"
)

type StringData struct {
	Strings []string
}

func (d StringData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d.Strings, w)
}

func (d StringData) Encode(format string, w io.Writer) error {
	_, err := w.Write(JsonMarshal(d.Strings))
	return err
}

func (d StringData) Type() DataType {
	return StringType
}

func (d StringData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d StringData) GetMetadata() interface{} {
	return nil
}

// SliceData
func (d StringData) Length() int {
	return len(d.Strings)
}
func (d StringData) Slice(i, j int) Data {
	return StringData{Strings: d.Strings[i:j]}
}
func (d StringData) Append(other Data) Data {
	return StringData{
		Strings: append(d.Strings, other.(StringData).Strings...),
	}
}

func (d StringData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[StringType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data StringData
			if err := ReadJsonData(r, &data.Strings); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var data StringData
			ReadJSONFile(fname, &data.Strings)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			var data StringData
			JsonUnmarshal(bytes, &data.Strings)
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "json", "", nil
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: StringData{}}
		},
		ChunkType: StringType,
	}
}
