package spatialflow_partition

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"
	"log"
	"fmt"
	"math"
	"github.com/paulmach/go.geojson"
	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
	geocoords "github.com/mitroadmaps/gomapinfer/googlemaps"
)

// We just use grid partition now. Later on we will add support for rectangle partition.
// Parameters for grid partition:
// - URL, similar to the URL in 'make_geoimage'
// - Zoom Level, e.g.,similar to 'make_geoimage'
// - ROI Buffer, e.g., 128 pixels
type Params struct {
	URL string
	Zoom int
	Buffer int
}

// TODO: Should split this into multiple tasks so that they can run in parallel. 
func SpatialFlowPartition(url string, outputDataset skyhook.Dataset, task skyhook.ExecTask, params Params) error {
	// Parameters (should be assigned from UI)
	var zoom int = params.Zoom
	var shapeBuffer int = params.Buffer
	mapurl := params.URL
	
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
	log.Printf("[spatialflow_partition] got %d geometries from GeoJSON files", len(geometries))

	// Compute the bounding box 
	var geometriesBBox gomapinfer.Rectangle = gomapinfer.EmptyRectangle
	handlePointBBox := func(coordinate []float64) {
		p := gomapinfer.Point{coordinate[0], coordinate[1]}
		geometriesBBox = geometriesBBox.Extend(p)
	}

	handleLineStringBBox := func(coordinates [][]float64) {
		for _, coordinate := range coordinates {
			p := gomapinfer.Point{coordinate[0], coordinate[1]}
			geometriesBBox = geometriesBBox.Extend(p)
		}
	}
	handlePolygonBBox := func(coordinates [][][]float64) {
		// We do not support holes yet, so just use coordinates[0].
		// coordinates[0] is the exterior ring while coordinates[1:] specify
		// holes in the polygon that should be excluded.
		for _, coordinate := range coordinates[0] {
			p := gomapinfer.Point{coordinate[0], coordinate[1]}
			geometriesBBox = geometriesBBox.Extend(p)
		}
	}

	for _, g := range geometries {
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
	}

	// Grid partition
	startTile := geocoords.LonLatToMapboxTile(geometriesBBox.Min, zoom)
	endTile := geocoords.LonLatToMapboxTile(geometriesBBox.Max, zoom)
	if startTile[0] > endTile[0] {
		startTile[0], endTile[0] = endTile[0], startTile[0]
	}
	if startTile[1] > endTile[1] {
		startTile[1], endTile[1] = endTile[1], startTile[1]
	}
	log.Printf("[spatialflow_partition] checking candidate tiles from %v to %v", startTile, endTile)
	buffer := float64(shapeBuffer) / 256.0 
	bufferTiles := int(math.Ceil(buffer))
	corner1 := gomapinfer.Point{0.0 - buffer, 0.0 - buffer}
	corner2 := gomapinfer.Point{1.0 + buffer, 0.0 - buffer}
	corner3 := gomapinfer.Point{1.0 + buffer, 1.0 + buffer}
	corner4 := gomapinfer.Point{0.0 - buffer, 1.0 + buffer}
	corners := [4]gomapinfer.Point{corner1, corner2, corner3, corner4}

	var total_tiles int = 0
	var kept_tiles int = 0

	// TODO: This is a O(n^2) implementation. Should improve it by using spatial index.
	for i := startTile[0] - bufferTiles; i <= endTile[0] + bufferTiles; i++ {
		for j := startTile[1] - bufferTiles; j <= endTile[1] + bufferTiles; j++ {
			total_tiles += 1

			p1 := geocoords.MapboxToLonLat(gomapinfer.Point{0,0}, zoom, [2]int{i,j})
			p2 := geocoords.MapboxToLonLat(gomapinfer.Point{0,0}, zoom, [2]int{i+1,j+1})
			
			toRelativePixelCoordinate := func(coordinate []float64) gomapinfer.Point {
				var point gomapinfer.Point
				point.X = (coordinate[0] - p1.X) / (p2.X - p1.X)
				point.Y = (coordinate[1] - p1.Y) / (p2.Y - p1.Y)
				return point 
			}

			isOverlapped := false

			// Check if the tile overlaps (consider the buffer) with the ROI (different shapes)
			handlePoint := func(coordinate []float64) {
				p := toRelativePixelCoordinate(coordinate)
				if p.X >= -buffer && p.X <= 1.0 + buffer && p.Y >= -buffer && p.Y <= 1.0 + buffer {
					isOverlapped = true
				}  
			}

			handleLineString := func(coordinates [][]float64) {
				for _, coordinate := range coordinates {
					p := toRelativePixelCoordinate(coordinate)
					if p.X >= -buffer && p.X <= 1.0 + buffer && p.Y >= -buffer && p.Y <= 1.0 + buffer {
						isOverlapped = true
					}  
				}
				if isOverlapped {
					return 
				}
				for ind, _ := range coordinates {
					if ind == len(coordinates)-1 {
						break
					}

					p1 := toRelativePixelCoordinate(coordinates[ind])
					p2 := toRelativePixelCoordinate(coordinates[ind+1])
					
					segment := gomapinfer.Segment{p1,p2}
					if segment.Intersection(gomapinfer.Segment{corner1, corner2}) != nil {
						isOverlapped = true
						return 
					}
					if segment.Intersection(gomapinfer.Segment{corner2, corner3}) != nil {
						isOverlapped = true
						return 
					}
					if segment.Intersection(gomapinfer.Segment{corner3, corner4}) != nil {
						isOverlapped = true
						return 
					}
					if segment.Intersection(gomapinfer.Segment{corner4, corner1}) != nil {
						isOverlapped = true
						return 
					}
				}
			}

			handlePolygon := func(coordinates [][][]float64) {
				// We do not support holes yet, so just use coordinates[0].
				// coordinates[0] is the exterior ring while coordinates[1:] specify
				// holes in the polygon that should be excluded.
				handleLineString(coordinates[0])

				if isOverlapped {
					return
				}
				
				var polygon gomapinfer.Polygon
				for _, coordinate := range coordinates[0] {
					p := toRelativePixelCoordinate(coordinate)
					polygon = append(polygon, p)
				}
				
				for _, corner := range corners {
					if polygon.Contains(corner) {
						isOverlapped = true
						return 
					}
				}

			}
			
			for _, g := range geometries {
				if isOverlapped {
					break
				}

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
			
			if !isOverlapped {
				continue
			}

			// If the current tile overlaps with the ROI, store it in a dataset
			log.Printf("[spatialflow_partition] create tile %s", fmt.Sprintf("%d_%d_%d", zoom, i, j))
			outputData := skyhook.GeoImageData{
				Metadata: skyhook.GeoImageMetadata{
					ReferenceType: "webmercator",
					Zoom: zoom,
					X: i,
					Y: j,
					Scale: 256,
					Width: 256,
					Height: 256,
					SourceType: "url",
					URL: mapurl,
				},
			}
			err := exec_ops.WriteItem(url, outputDataset, fmt.Sprintf("%d_%d_%d", zoom, i, j), outputData)
			if err != nil {
				return err
			}
			kept_tiles += 1
			
		}
	}

	log.Printf("[spatialflow_partition] found %d tiles overlapping with the ROI from %d tiles", kept_tiles, total_tiles)
				

	return nil
}


func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "spatialflow_partition",
			Name: "SpatialFlow Partition",
			Description: "Partition a ROI (Geojson) into rectangular regions (Geo-Image)",
		},
		Inputs: []skyhook.ExecInput{
			{Name: "geojson", DataTypes: []skyhook.DataType{skyhook.GeoJsonType}},
		},
		Outputs: []skyhook.ExecOutput{{Name: "geoimages", DataType: skyhook.GeoImageType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return nil, err
			}

			applyFunc := func(task skyhook.ExecTask) error {
				return SpatialFlowPartition(url, node.OutputDatasets["geoimages"], task, params) 
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
