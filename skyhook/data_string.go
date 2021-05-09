package skyhook

import (
	"encoding/json"
)

type StringJsonSpec struct {}

func (s StringJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

func (s StringJsonSpec) Decode(dec *json.Decoder, n int) (interface{}, error) {
	var data []string
	for i := 0; (i < n || n == -1) && dec.More(); i++ {
		var cur string
		err := dec.Decode(&cur)
		if err != nil {
			return nil, err
		}
		data = append(data, cur)
	}
	return data, nil
}

func (s StringJsonSpec) Encode(enc *json.Encoder, data interface{}) error {
	for _, cur := range data.([]string) {
		err := enc.Encode(cur)
		if err != nil {
			return err
		}
	}
	return nil
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
