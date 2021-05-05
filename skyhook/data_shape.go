package skyhook

import (
	"encoding/json"
)

type ShapeMetadata struct {
	CanvasDims [2]int `json:",omitempty"`
	Categories []string `json:",omitempty"`
}

func (m ShapeMetadata) Update(other DataMetadata) DataMetadata {
	other_ := other.(ShapeMetadata)
	if other_.CanvasDims[0] > 0 {
		m.CanvasDims = other_.CanvasDims
	}
	if len(other_.Categories) > 0 {
		m.Categories = other_.Categories
	}
	return m
}

// Shape types.
type TypeOfShape string
const (
	PointShape TypeOfShape = "point"
	LineShape = "line"
	PolyLineShape = "polyline"
	BoxShape = "box"
	PolygonShape = "polygon"
)

type Shape struct {
	Type TypeOfShape
	Points [][2]int

	// Optional metadata
	Category string `json:",omitempty"`
	TrackID int `json:",omitempty"`
	Metadata map[string]string `json:",omitempty"`
}

func (shp Shape) Bounds() [4]int {
	sx := shp.Points[0][0]
	sy := shp.Points[0][1]
	ex := shp.Points[0][0]
	ey := shp.Points[0][1]
	for _, p := range shp.Points {
		if p[0] < sx {
			sx = p[0]
		}
		if p[0] > ex {
			ex = p[0]
		}
		if p[1] < sy {
			sy = p[1]
		}
		if p[1] > ey {
			ey = p[1]
		}
	}
	return [4]int{sx, sy, ex, ey}
}

type ShapeJsonSpec struct {}

func (s ShapeJsonSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	if rawMetadata == "" {
		return ShapeMetadata{}
	}
	var m ShapeMetadata
	JsonUnmarshal([]byte(rawMetadata), &m)
	return m
}

func (s ShapeJsonSpec) Decode(dec *json.Decoder, n int) (interface{}, error) {
	var data [][]Shape
	for i := 0; (i < n || n == -1) && dec.More(); i++ {
		var cur []Shape
		err := dec.Decode(&cur)
		if err != nil {
			return nil, err
		}
		data = append(data, cur)
	}
	return data, nil
}

func (s ShapeJsonSpec) Encode(enc *json.Encoder, data interface{}) error {
	for _, cur := range data.([][]Shape) {
		err := enc.Encode(cur)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s ShapeJsonSpec) GetEmptyData() (data interface{}) {
	return [][]Shape{}
}

func (s ShapeJsonSpec) GetEmptyMetadata() (metadata DataMetadata) {
	return ShapeMetadata{}
}

func (s ShapeJsonSpec) Length(data interface{}) int {
	return len(data.([][]Shape))
}
func (s ShapeJsonSpec) Append(data interface{}, more interface{}) interface{} {
	return append(data.([][]Shape), more.([][]Shape)...)
}
func (s ShapeJsonSpec) Slice(data interface{}, i int, j int) interface{} {
	return data.([][]Shape)[i:j]
}

func init() {
	DataSpecs[ShapeType] = SequenceJsonDataImpl{ShapeJsonSpec{}}
}
