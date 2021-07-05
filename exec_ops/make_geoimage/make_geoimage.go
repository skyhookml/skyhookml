package make_geoimage

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
	geocoords "github.com/mitroadmaps/gomapinfer/googlemaps"
	"github.com/paulmach/go.geojson"

	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime"
)

const GeoImageScale int = 256

type Params struct {
	Source struct {
		// Either "url" or "dataset"
		Mode string
		// Only set if Mode is "url".
		URL string
		Zoom int
	}
	// Specifies how we should determine what images to capture.
	// One of "dense" or "geojson".
	CaptureMode string
	// Image dimensions.
	// If CaptureMode is "geojson" and ImageDims are 0, then width/height is based on
	// the size of the GeoJSON object.
	ImageDims [2]int
	// If CaptureMode is "dense", specifies the bounding box to densely cover.
	Bbox [4]float64
	// If CaptureMode is GeoJSON, specifies how we should build images around GeoJSON objects.
	// One of "centered-all", "centered-disjoint", or "tiles".
	ObjectMode string
	// If ObjectMode == "tiles", Buffer is padding around GeoJSON objects that should be covered by tiles.
	Buffer int
	// Whether we should download images immediately or defer until the image is needed.
	Materialize bool
}

type TaskMetadata struct {
	GeoImageMetadata skyhook.GeoImageMetadata
}

// Get GeoImageMetadata for params.CaptureMode=="dense".
// In the metadata, only Zoom, X, Y, Offset, Width, and Height are set.
func GetDenseMetadatas(params Params) []skyhook.GeoImageMetadata {
	startTile := geocoords.LonLatToMapboxTile(gomapinfer.Point{params.Bbox[0], params.Bbox[1]}, params.Source.Zoom)
	endTile := geocoords.LonLatToMapboxTile(gomapinfer.Point{params.Bbox[2], params.Bbox[3]}, params.Source.Zoom)
	if startTile[0] > endTile[0] {
		startTile[0], endTile[0] = endTile[0], startTile[0]
	}
	if startTile[1] > endTile[1] {
		startTile[1], endTile[1] = endTile[1], startTile[1]
	}
	log.Printf("[make_geoimage] adding tiles from %v to %v", startTile, endTile)
	var metadatas []skyhook.GeoImageMetadata
	for i := startTile[0]; i <= endTile[0]; i++ {
		for j := startTile[1]; j <= endTile[1]; j++ {
			metadatas = append(metadatas, skyhook.GeoImageMetadata{
				Zoom: params.Source.Zoom,
				X: i,
				Y: j,
				Width: GeoImageScale,
				Height: GeoImageScale,
			})
		}
	}
	return metadatas
}

// Get GeoImageMetadata for params.CaptureMode=="geojson" with ObjectMode=="centered-all" or "centered-disjoint".
// This creates images that are centered at each geojson object.
// In the metadata, only Zoom, X, Y, Offset, Width, and Height are set.
func GetGeojsonCenteredMetadatas(params Params, geometries []*geojson.Geometry) []skyhook.GeoImageMetadata {
	disjoint := params.ObjectMode == "centered-disjoint"
	zoom := params.Source.Zoom

	// We keep track of all Web-Mercator tiles that the images so far have intersected.
	// This way we can avoid adding overlapping tiles, if disjoint is set.
	seen := make(map[[2]int]bool)

	var metadatas []skyhook.GeoImageMetadata

	for _, g := range geometries {
		bbox := skyhook.GetGeometryBbox(g)
		var metadata skyhook.GeoImageMetadata

		if params.ImageDims == [2]int{0, 0} {
			// The dimensions are based on the bbox size.
			tile := geocoords.LonLatToMapboxTile(bbox.Min, zoom)
			offset1 := geocoords.LonLatToMapbox(bbox.Min, zoom, tile)
			offset2 := geocoords.LonLatToMapbox(bbox.Max, zoom, tile)
			// Make offset1 the smaller of the two.
			if offset1.X > offset2.X {
				offset1.X, offset2.X = offset2.X, offset1.X
			}
			if offset1.Y > offset2.Y {
				offset1.Y, offset2.Y = offset2.Y, offset1.Y
			}
			metadata = skyhook.GeoImageMetadata{
				Zoom: zoom,
				X: tile[0],
				Y: tile[1],
				Offset: [2]int{int(offset1.X), int(offset1.Y)},
				Width: int(offset2.X-offset1.X),
				Height: int(offset2.Y-offset1.Y),
			}
		} else {
			// Fixed size bbox.
			bboxCenter := bbox.Center()
			centerTile := geocoords.LonLatToMapboxTile(bboxCenter, zoom)
			centerOffset := geocoords.LonLatToMapbox(bboxCenter, zoom, centerTile)
			metadata = skyhook.GeoImageMetadata{
				Zoom: zoom,
				X: centerTile[0],
				Y: centerTile[1],
				Offset: [2]int{int(centerOffset.X) - params.ImageDims[0]/2, int(centerOffset.Y) - params.ImageDims[1]/2},
				Width: params.ImageDims[0],
				Height: params.ImageDims[1],
			}
		}

		if disjoint {
			// Compute tile offsets from metadata.X/Y where the image starts and ends.
			// This way we can get list of WebMercator tiles that intersect this image.
			startOffset := [2]int{
				skyhook.FloorDiv(metadata.Offset[0], GeoImageScale),
				skyhook.FloorDiv(metadata.Offset[1], GeoImageScale),
			}
			endOffset := [2]int{
				skyhook.FloorDiv(metadata.Offset[0]+metadata.Width-1, GeoImageScale),
				skyhook.FloorDiv(metadata.Offset[1]+metadata.Height-1, GeoImageScale),
			}
			var needed [][2]int
			for i := startOffset[0]; i <= endOffset[0]; i++ {
				for j := startOffset[1]; j <= endOffset[1]; j++ {
					needed = append(needed, [2]int{i, j})
				}
			}

			skip := false
			for _, tile := range needed {
				if seen[tile] {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			for _, tile := range needed {
				seen[tile] = true
			}
		}

		metadatas = append(metadatas, metadata)
	}

	return metadatas
}

// Get GeoImageMetadata for params.CaptureMode=="geojson" with ObjectMode=="tiles".
// This captures WebMercator tiles of a fixed size that intersect a GeoJSON object.
// In the metadata, only Zoom, X, Y, Offset, Width, and Height are set.
func GetGeojsonTilesMetadatas(params Params, geometries []*geojson.Geometry) []skyhook.GeoImageMetadata {
	var zoom int = params.Source.Zoom
	var shapeBuffer int = params.Buffer

	// Compute the bounding box of all the geometries.
	var geometriesBBox gomapinfer.Rectangle = gomapinfer.EmptyRectangle

	for _, g := range geometries {
		bbox := skyhook.GetGeometryBbox(g)
		geometriesBBox = geometriesBBox.Extend(bbox.Min).Extend(bbox.Max)
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

	var metadatas []skyhook.GeoImageMetadata

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

				if isOverlapped {
					break
				}
			}

			// If the current tile doesn't overlap with the ROI, skip it.
			if !isOverlapped {
				continue
			}

			metadatas = append(metadatas, skyhook.GeoImageMetadata{
				Zoom: zoom,
				X: i,
				Y: j,
				Width: GeoImageScale,
				Height: GeoImageScale,
			})
			kept_tiles += 1
		}
	}

	log.Printf("[spatialflow_partition] found %d tiles overlapping with the ROI from %d tiles", kept_tiles, total_tiles)
	return metadatas
}

func GetGeojsonMetadatas(params Params, allItems map[string][][]skyhook.Item) ([]skyhook.GeoImageMetadata, error) {
	// Load all GeoJSON geometries.
	// Note that we do this in the GetTasks call, since we want to parallelize the
	// image download/extraction execution (in the case that params.Materialize is set).
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
	for _, itemList := range allItems["geojson"] {
		for _, item := range itemList {
			data, _, err := item.LoadData()
			if err != nil {
				return nil, err
			}
			addFeatures(data.(*geojson.FeatureCollection))
		}
	}

	if params.ObjectMode == "centered-all" || params.ObjectMode == "centered-disjoint" {
		return GetGeojsonCenteredMetadatas(params, geometries), nil
	} else if params.ObjectMode == "tiles" {
		return GetGeojsonTilesMetadatas(params, geometries), nil
	}
	return nil, fmt.Errorf("unknown object mode %s", params.ObjectMode)
}

func GetTasks(node skyhook.Runnable, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
	var params Params
	err := json.Unmarshal([]byte(node.Params), &params)
	if err != nil {
		return nil, fmt.Errorf("node has not been configured: %v", err)
	}
	var metadatas []skyhook.GeoImageMetadata
	if params.CaptureMode == "dense" {
		metadatas = GetDenseMetadatas(params)
	} else if params.CaptureMode == "geojson" {
		var err error
		metadatas, err = GetGeojsonMetadatas(params, allItems)
		if err != nil {
			return nil, err
		}
	}
	var tasks []skyhook.ExecTask
	for i, metadata := range metadatas {
		var key string
		if metadata.Offset == [2]int{0, 0} && metadata.Width == GeoImageScale && metadata.Height == GeoImageScale {
			key = fmt.Sprintf("%d_%d_%d", metadata.Zoom, metadata.X, metadata.Y)
		} else {
			key = fmt.Sprintf("%d", i)
		}
		metadata.ReferenceType = "webmercator"
		metadata.Scale = 256
		metadata.SourceType = "url"
		metadata.URL = params.Source.URL
		tasks = append(tasks, skyhook.ExecTask{
			Key: key,
			Metadata: string(skyhook.JsonMarshal(TaskMetadata{metadata})),
		})
	}
	return tasks, nil
}

type Op struct {
	URL string
	Params Params
	Dataset skyhook.Dataset
}

func (e *Op) Parallelism() int {
	return runtime.NumCPU()
}

func (e *Op) Apply(task skyhook.ExecTask) error {
	// TODO: add support for Materialize.
	var metadata TaskMetadata
	skyhook.JsonUnmarshal([]byte(task.Metadata), &metadata)
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, nil, metadata.GeoImageMetadata)
}

func (e *Op) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "make_geoimage",
			Name: "Make Geo-Image Dataset",
			Description: "Create a Geo-Image dataset by fetching tiles from a URL",
		},
		GetInputs: func(rawParams string) []skyhook.ExecInput {
			var params Params
			err := json.Unmarshal([]byte(rawParams), &params)
			if err != nil {
				// can't do anything if node isn't configured yet
				return nil
			}
			if params.CaptureMode == "geojson" {
				return []skyhook.ExecInput{{
					Name: "geojson",
					DataTypes: []skyhook.DataType{skyhook.GeoJsonType},
					Variable: true,
				}}
			}
			return nil
		},
		Outputs: []skyhook.ExecOutput{{Name: "geoimages", DataType: skyhook.GeoImageType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: GetTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			var params Params
			if err := exec_ops.DecodeParams(node, &params, false); err != nil {
				return nil, err
			}
			return &Op{
				URL: url,
				Params: params,
				Dataset: node.OutputDatasets["geoimages"],
			}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
