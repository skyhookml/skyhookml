package skyhook

import (
	"io"
	"io/ioutil"
)

type FloatData struct {
	Floats [][]float64
}

func (d FloatData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d.Floats, w)
}

func (d FloatData) Encode(format string, w io.Writer) error {
	_, err := w.Write(JsonMarshal(d.Floats))
	return err
}

func (d FloatData) Type() DataType {
	return FloatsType
}

func (d FloatData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d FloatData) GetMetadata() interface{} {
	return nil
}

// SliceData
func (d FloatData) Length() int {
	return len(d.Floats)
}
func (d FloatData) Slice(i, j int) Data {
	return FloatData{Floats: d.Floats[i:j]}
}
func (d FloatData) Append(other Data) Data {
	return FloatData{
		Floats: append(d.Floats, other.(FloatData).Floats...),
	}
}

func (d FloatData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[FloatsType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data FloatData
			if err := ReadJsonData(r, &data.Floats); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var data FloatData
			ReadJSONFile(fname, &data.Floats)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			var data FloatData
			JsonUnmarshal(bytes, &data.Floats)
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "json", "", nil
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: FloatData{}}
		},
		ChunkType: FloatsType,
	}
}
