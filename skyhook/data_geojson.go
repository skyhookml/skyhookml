package skyhook

import (
	"io"
	"io/ioutil"

	"github.com/paulmach/go.geojson"
)

type GeoJsonData struct {
	Collection *geojson.FeatureCollection
}

func (d GeoJsonData) EncodeStream(w io.Writer) error {
	return WriteJsonData(d.Collection, w)
}

func (d GeoJsonData) Encode(format string, w io.Writer) error {
	_, err := w.Write(JsonMarshal(d.Collection))
	return err
}

func (d GeoJsonData) Type() DataType {
	return GeoJsonType
}

func (d GeoJsonData) GetDefaultExtAndFormat() (string, string) {
	return "json", "json"
}

func (d GeoJsonData) GetMetadata() interface{} {
	return nil
}

func init() {
	DataImpls[GeoJsonType] = DataImpl{
		DecodeStream: func(r io.Reader) (Data, error) {
			var data GeoJsonData
			if err := ReadJsonData(r, &data.Collection); err != nil {
				return nil, err
			}
			return data, nil
		},
		DecodeFile: func(format string, metadataRaw string, fname string) (Data, error) {
			var data GeoJsonData
			ReadJSONFile(fname, &data.Collection)
			return data, nil
		},
		Decode: func(format string, metadataRaw string, r io.Reader) (Data, error) {
			bytes, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			var data GeoJsonData
			JsonUnmarshal(bytes, &data.Collection)
			return data, nil
		},
		GetDefaultMetadata: func(fname string) (format string, metadataRaw string, err error) {
			return "json", "", nil
		},
	}
}
