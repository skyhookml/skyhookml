package geojson_to_shape

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"log"

	"github.com/paulmach/go.geojson"
	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
)

func GeoJsonToShape(url string, outputDataset skyhook.Dataset, task skyhook.ExecTask) error {
	// Load all GeoJSON geometries.
	var geometries []*geojson.Geometry
	addFeatures := func(collection *geojson.FeatureCollection) {
		var q []*geojson.Geometry
		for _, feature := range collection.Features {
			if feature.Geometry == nil {
				continue
			}
			q = append(q, feature.Geometry)
		}
		for len(q) > 0 {
			geometry := q[len(q)-1]
			q = q[0:len(q)-1]
			if geometry.Type != geojson.GeometryCollection {
				geometries = append(geometries, geometry)
				continue
			}
			// collection geometry, need to add all its children
			q = append(q, geometry.Geometries...)
		}
	}
	for _, item := range task.Items["geojson"][0] {
		data, err := item.LoadData()
		if err != nil {
			return err
		}
		addFeatures(data.(skyhook.GeoJsonData).Collection)
	}
	log.Printf("[geojson_to_shape] got %d geometries from GeoJSON files", len(geometries))

	// Loop over the images and find the geometries that intersect each one.
	// For now we do O(n^2) loop but later we could create a spatial index.
	for _, item := range task.Items["images"][0] {
		var metadata skyhook.GeoImageMetadata
		skyhook.JsonUnmarshal([]byte(item.Metadata), &metadata)
		bbox := metadata.GetBbox()
		rect := bbox.Rect()
		dims := [2]int{metadata.Width, metadata.Height}

		fromGeo := func(coordinate []float64) [2]int {
			p := bbox.FromGeo([2]float64{coordinate[0], coordinate[1]})
			return [2]int{
				int(p[0]*float64(dims[0])),
				int(p[1]*float64(dims[1])),
			}
		}

		var shapes []skyhook.Shape

		handlePoint := func(coordinate []float64) {
			p := gomapinfer.Point{coordinate[0], coordinate[1]}
			if !rect.Contains(p) {
				return
			}
			shapes = append(shapes, skyhook.Shape{
				Type: skyhook.PointShape,
				Points: [][2]int{fromGeo(coordinate)},
			})
		}

		handleLineString := func(coordinates [][]float64) {
			bounds := gomapinfer.EmptyRectangle
			for _, coordinate := range coordinates {
				p := gomapinfer.Point{coordinate[0], coordinate[1]}
				bounds = bounds.Extend(p)
			}
			if !rect.Intersects(bounds) {
				return
			}
			points := make([][2]int, len(coordinates))
			for i := range points {
				points[i] = fromGeo(coordinates[i])
			}
			shapes = append(shapes, skyhook.Shape{
				Type: skyhook.PolyLineShape,
				Points: points,
			})
		}

		handlePolygon := func(coordinates [][][]float64) {
			// We do not support holes yet, so just use coordinates[0].
			// coordinates[0] is the exterior ring while coordinates[1:] specify
			// holes in the polygon that should be excluded.
			bounds := gomapinfer.EmptyRectangle
			for _, coordinate := range coordinates[0] {
				p := gomapinfer.Point{coordinate[0], coordinate[1]}
				bounds = bounds.Extend(p)
			}
			if !rect.Intersects(bounds) {
				return
			}
			points := make([][2]int, len(coordinates[0]))
			for i := range points {
				points[i] = fromGeo(coordinates[0][i])
			}
			shapes = append(shapes, skyhook.Shape{
				Type: skyhook.PolygonShape,
				Points: points,
			})
		}

		for _, g := range geometries {
			if g.Type == geojson.GeometryPoint {
				handlePoint(g.Point)
			} else if g.Type == geojson.GeometryMultiPoint {
				for _, coordinate := range g.MultiPoint {
					handlePoint(coordinate)
				}
			} else if g.Type == geojson.GeometryLineString {
				handleLineString(g.LineString)
			} else if g.Type == geojson.GeometryMultiLineString {
				for _, coordinates := range g.MultiLineString {
					handleLineString(coordinates)
				}
			} else if g.Type == geojson.GeometryPolygon {
				handlePolygon(g.Polygon)
			} else if g.Type == geojson.GeometryMultiPolygon {
				for _, coordinates := range g.MultiPolygon {
					handlePolygon(coordinates)
				}
			}
		}

		shapeData := skyhook.ShapeData{
			Shapes: [][]skyhook.Shape{shapes},
			Metadata: skyhook.ShapeMetadata{
				CanvasDims: dims,
			},
		}
		err := exec_ops.WriteItem(url, outputDataset, item.Key, shapeData)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "geojson_to_shape",
			Name: "GeoJSON to Shape",
			Description: "Convert from GeoJSON to Shape type given a Geo-Image dataset",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "geojson", DataTypes: []skyhook.DataType{skyhook.GeoJsonType}},
			{Name: "images", DataTypes: []skyhook.DataType{skyhook.GeoImageType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "shapes", DataType: skyhook.ShapeType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("merged"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				return GeoJsonToShape(url, node.OutputDatasets["shapes"], task)
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
