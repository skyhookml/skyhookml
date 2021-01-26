package skyhook

import (
	"io"
	"io/ioutil"
)

type ShapeMetadata struct {
	CanvasDims [2]int
	Categories []string `json:",omitempty"`
}

type Shape struct {
	Type string
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

type ShapeData struct {
	Shapes [][]Shape
	Metadata ShapeMetadata
}

func (d ShapeData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d, w)
}

func (d ShapeData) Encode(format string, w io.Writer) error {
	_, err := w.Write(JsonMarshal(d.Shapes))
	return err
}

func (d ShapeData) Type() DataType {
	return ShapeType
}

func (d ShapeData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d ShapeData) GetMetadata() interface{} {
	return d.Metadata
}

// SliceData
func (d ShapeData) Length() int {
	return len(d.Shapes)
}
func (d ShapeData) Slice(i, j int) Data {
	return ShapeData{
		Metadata: d.Metadata,
		Shapes: d.Shapes[i:j],
	}
}
func (d ShapeData) Append(other Data) Data {
	other_ := other.(ShapeData)
	return ShapeData{
		Metadata: other_.Metadata,
		Shapes: append(d.Shapes, other_.Shapes...),
	}
}

func (d ShapeData) Reader() DataReader {
	return &SliceReader{Data: d}
}

func init() {
	DataImpls[ShapeType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data ShapeData
			if err := ReadJsonData(r, &data); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var metadata ShapeMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			data := ShapeData{Metadata: metadata}
			ReadJSONFile(fname, &data.Shapes)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			var metadata ShapeMetadata
			JsonUnmarshal([]byte(metadataRaw), &metadata)

			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			data := ShapeData{Metadata: metadata}
			JsonUnmarshal(bytes, &data.Shapes)
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "json", "", nil
		},
		Builder: func() ChunkBuilder {
			return &SliceBuilder{Data: ShapeData{}}
		},
		ChunkType: ShapeType,
	}
}
