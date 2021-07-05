package skyhook

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/paulmach/go.geojson"
	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
)

type GeoJsonData struct {
	Collection *geojson.FeatureCollection
}

func GetGeometryBbox(g *geojson.Geometry) gomapinfer.Rectangle {
	var bbox gomapinfer.Rectangle = gomapinfer.EmptyRectangle

	handlePointBBox := func(coordinate []float64) {
		p := gomapinfer.Point{coordinate[0], coordinate[1]}
		bbox = bbox.Extend(p)
	}
	handleLineStringBBox := func(coordinates [][]float64) {
		for _, coordinate := range coordinates {
			p := gomapinfer.Point{coordinate[0], coordinate[1]}
			bbox = bbox.Extend(p)
		}
	}
	handlePolygonBBox := func(coordinates [][][]float64) {
		// We do not support holes yet, so just use coordinates[0].
		// coordinates[0] is the exterior ring while coordinates[1:] specify
		// holes in the polygon that should be excluded.
		for _, coordinate := range coordinates[0] {
			p := gomapinfer.Point{coordinate[0], coordinate[1]}
			bbox = bbox.Extend(p)
		}
	}

	if g.Type == geojson.GeometryPoint {
		handlePointBBox(g.Point)
	} else if g.Type == geojson.GeometryMultiPoint {
		for _, coordinate := range g.MultiPoint {
			handlePointBBox(coordinate)
		}
	} else if g.Type == geojson.GeometryLineString {
		handleLineStringBBox(g.LineString)
	} else if g.Type == geojson.GeometryMultiLineString {
		for _, coordinates := range g.MultiLineString {
			handleLineStringBBox(coordinates)
		}
	} else if g.Type == geojson.GeometryPolygon {
		handlePolygonBBox(g.Polygon)
	} else if g.Type == geojson.GeometryMultiPolygon {
		for _, coordinates := range g.MultiPolygon {
			handlePolygonBBox(coordinates)
		}
	}

	return bbox
}

type GeoJsonDataSpec struct{}

func (s GeoJsonDataSpec) DecodeMetadata(rawMetadata string) DataMetadata {
	return NoMetadata{}
}

func (s GeoJsonDataSpec) ReadStream(r io.Reader) (interface{}, error) {
	var data *geojson.FeatureCollection
	if err := ReadJsonData(r, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s GeoJsonDataSpec) WriteStream(data interface{}, w io.Writer) error {
	if err := WriteJsonData(data, w); err != nil {
		return err
	}
	return nil
}

func (s GeoJsonDataSpec) Read(format string, metadata DataMetadata, r io.Reader) (interface{}, error) {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var data *geojson.FeatureCollection
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s GeoJsonDataSpec) Write(data interface{}, format string, metadata DataMetadata, w io.Writer) error {
	_, err := w.Write(JsonMarshal(data))
	return err
}

func (s GeoJsonDataSpec) GetDefaultExtAndFormat(data interface{}, metadata DataMetadata) (ext string, format string) {
	return "json", "json"
}

func init() {
	DataSpecs[GeoJsonType] = GeoJsonDataSpec{}
}
