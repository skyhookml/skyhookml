package skyhook

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/paulmach/go.geojson"
)

type GeoJsonData struct {
	Collection *geojson.FeatureCollection
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
