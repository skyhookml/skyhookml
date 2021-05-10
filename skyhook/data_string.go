package skyhook

import (
	"encoding/json"
)

type StringJsonSpec struct {}

func (s StringJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

func (s StringJsonSpec) DecodeData(bytes []byte) (interface{}, error) {
	var data []string
	err := json.Unmarshal(bytes, &data)
	return data, err
}

func (s StringJsonSpec) GetEmptyMetadata() (metadata DataMetadata) {
	return NoMetadata{}
}

func (s StringJsonSpec) Length(data interface{}) int {
	return len(data.([]string))
}
func (s StringJsonSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([]string), more.([]string)...)
}
func (s StringJsonSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([]string)[i:j]
}

func init() {
	DataSpecs[StringType] = SequenceJsonDataImpl{StringJsonSpec{}}
}
