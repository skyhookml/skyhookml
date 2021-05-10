package skyhook

import (
	"encoding/json"
)

type FloatJsonSpec struct {}

func (s FloatJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

func (s FloatJsonSpec) DecodeData(bytes []byte) (interface{}, error) {
	var data [][]float64
	err := json.Unmarshal(bytes, &data)
	return data, err
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
