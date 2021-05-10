package skyhook

import (
	"encoding/json"
	"math"
)

type DetectionMetadata struct {
	CanvasDims [2]int `json:",omitempty"`
	Categories []string `json:",omitempty"`
}

func (m DetectionMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(DetectionMetadata)
	if other_.CanvasDims[0] > 0 {
		m.CanvasDims = other_.CanvasDims
	}
	if len(other_.Categories) > 0 {
		m.Categories = other_.Categories
	}
	return m
}

type Detection struct {
	Left int
	Top int
	Right int
	Bottom int

	// Optional metadata
	Category string `json:",omitempty"`
	TrackID int `json:",omitempty"`
	Score float64 `json:",omitempty"`
	Metadata map[string]string `json:",omitempty"`
}

func (d Detection) CenterDistance(other Detection) float64 {
	dx := (d.Left+d.Right-other.Left-other.Right)/2
	dy := (d.Top+d.Bottom-other.Top-other.Bottom)/2
	return math.Sqrt(float64(dx*dx+dy*dy))
}

func (d Detection) Rescale(origDims [2]int, newDims [2]int) Detection {
	copy := d
	copy.Left = copy.Left * newDims[0] / origDims[0]
	copy.Right = copy.Right * newDims[0] / origDims[0]
	copy.Top = copy.Top * newDims[1] / origDims[1]
	copy.Bottom = copy.Bottom * newDims[1] / origDims[1]
	return copy
}

type DetectionJsonSpec struct {}

func (s DetectionJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return DetectionMetadata{}
	}
	var m DetectionMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

func (s DetectionJsonSpec) DecodeData(bytes []byte) (interface{}, error) {
	var data [][]Detection
	err := json.Unmarshal(bytes, &data)
	return data, err
}

func (s DetectionJsonSpec) GetEmptyMetadata() (metadata DataMetadata) {
	return DetectionMetadata{}
}

func (s DetectionJsonSpec) Length(data interface{}) int {
	return len(data.([][]Detection))
}
func (s DetectionJsonSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([][]Detection), more.([][]Detection)...)
}
func (s DetectionJsonSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([][]Detection)[i:j]
}

func init() {
	DataSpecs[DetectionType] = SequenceJsonDataImpl{DetectionJsonSpec{}}
}
