package skyhook

import (
	"encoding/json"
)

type FloatJsonSpec struct {}

func (s FloatJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

func (s FloatJsonSpec) Decode(dec *json.Decoder, n int) (interface{}, error) {
	var data [][]float64
	for i := 0; (i < n || n == -1) && dec.More(); i++ {
		var cur []float64
		err := dec.Decode(&cur)
		if err != nil {
			return nil, err
		}
		data = append(data, cur)
	}
	return data, nil
}

func (s FloatJsonSpec) Encode(enc *json.Encoder, data interface{}) error {
	for _, cur := range data.([][]float64) {
		err := enc.Encode(cur)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s FloatJsonSpec) GetEmptyData() (data interface{}) {
	return [][]float64{}
}

func (s FloatJsonSpec) GetEmptyMetadata() (metadata DataMetadata) {
	return NoMetadata{}
}

func (s FloatJsonSpec) Length(data interface{}) int {
	return len(data.([][]float64))
}
func (s FloatJsonSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([][]float64), more.([][]float64)...)
}
func (s FloatJsonSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([][]float64)[i:j]
}

func init() {
	DataSpecs[FloatsType] = SequenceJsonDataImpl{FloatJsonSpec{}}
}
