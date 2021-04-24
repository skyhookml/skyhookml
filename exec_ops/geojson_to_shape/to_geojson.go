package geojson_to_shape

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"

	"github.com/paulmach/go.geojson"
)

func ShapeToGeoJson(url string, outputDataset skyhook.Dataset, task skyhook.ExecTask) error {
	// Helper function to convert shape to GeoJSON geometries given Geo-Image metadata.
	getGeometry := func(shape skyhook.Shape, canvasDims [2]int, geoMeta skyhook.GeoImageMetadata) (*geojson.Geometry, error) {
		// Convert shape.Points to geo coordinates.
		bbox := geoMeta.GetBbox()
		var coordinates [][]float64
		for _, intPoint := range shape.Points {
			p := [2]float64{float64(intPoint[0]), float64(intPoint[1])}
			p = [2]float64{
				p[0]/float64(canvasDims[0]),
				p[1]/float64(canvasDims[1]),
			}
			p = bbox.ToGeo(p)
			coordinates = append(coordinates, []float64{p[0], p[1], 0})
		}

		// Create geometry based on shape type.
		if shape.Type == skyhook.PointShape {
			return geojson.NewPointGeometry(coordinates[0]), nil
		} else if shape.Type == skyhook.PolyLineShape {
			return geojson.NewLineStringGeometry(coordinates), nil
		} else if shape.Type == skyhook.PolygonShape {
			return geojson.NewPolygonGeometry([][][]float64{coordinates}), nil
		} else {
			return nil, fmt.Errorf("cannot convert shape %s to GeoJSON", shape.Type)
		}
	}

	// Group corresponding Geo-Image and Shape items.
	// Map is key -> input name -> corresponding items under those names.
	grouped := exec_ops.GroupItems(task.Items)

	// Extract geometries from each pair of items, and add it to a FeatureCollection.
	fc := geojson.NewFeatureCollection()
	for _, items := range grouped {
		geoItem := items["images"][0]
		shapeItem := items["shapes"][0]
		var geoMeta skyhook.GeoImageMetadata
		skyhook.JsonUnmarshal([]byte(geoItem.Metadata), &geoMeta)
		shapeData_, err := shapeItem.LoadData()
		if err != nil {
			return err
		}
		shapeData := shapeData_.(skyhook.ShapeData)
		canvasDims := shapeData.Metadata.CanvasDims
		for _, shape := range shapeData.Shapes[0] {
			geometry, err := getGeometry(shape, canvasDims, geoMeta)
			if err != nil {
				return err
			}
			feature := geojson.NewFeature(geometry)
			fc.AddFeature(feature)
		}
	}

	// Write FeatureCollection to otuput dataset.
	return exec_ops.WriteItem(url, outputDataset, "geojson", skyhook.GeoJsonData{Collection: fc})
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "shape_to_geojson",
			Name: "Shape to GeoJSON",
			Description: "Convert from Shape to GeoJSON type given a Geo-Image dataset",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "shapes", DataTypes: []skyhook.DataType{skyhook.ShapeType}},
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.GeoImageType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "geojson", DataType: skyhook.GeoJsonType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("merged"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				return ShapeToGeoJson(url, node.OutputDatasets["geojson"], task)
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
