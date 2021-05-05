package make_geoimage

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	gomapinfer "github.com/mitroadmaps/gomapinfer/common"
	geocoords "github.com/mitroadmaps/gomapinfer/googlemaps"

	"encoding/json"
	"fmt"
	"log"
	"runtime"
)

type Params struct {
	URL string
	Zoom int
	Bbox [4]float64
}

type TaskMetadata struct {
	X int
	Y int
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
	var metadata TaskMetadata
	skyhook.JsonUnmarshal([]byte(task.Metadata), &metadata)
	outputMetadata := skyhook.GeoImageMetadata{
		ReferenceType: "webmercator",
		Zoom: e.Params.Zoom,
		X: metadata.X,
		Y: metadata.Y,
		Scale: 256,
		Width: 256,
		Height: 256,
		SourceType: "url",
		URL: e.Params.URL,
	}
	return exec_ops.WriteItem(e.URL, e.Dataset, task.Key, nil, outputMetadata)
}

func (e *Op) Close() {}

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "make_geoimage",
			Name: "Make Geo-Image Dataset",
			Description: "Create a Geo-Image dataset by fetching tiles from a URL",
		},
		Inputs: []skyhook.ExecInput{},
		Outputs: []skyhook.ExecOutput{{Name: "geoimages", DataType: skyhook.GeoImageType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: func(node skyhook.Runnable, allItems map[string][][]skyhook.Item) ([]skyhook.ExecTask, error) {
			var params Params
			err := json.Unmarshal([]byte(node.Params), &params)
			if err != nil {
				return nil, fmt.Errorf("node has not been configured: %v", err)
			}
			startTile := geocoords.LonLatToMapboxTile(gomapinfer.Point{params.Bbox[0], params.Bbox[1]}, params.Zoom)
			endTile := geocoords.LonLatToMapboxTile(gomapinfer.Point{params.Bbox[2], params.Bbox[3]}, params.Zoom)
			if startTile[0] > endTile[0] {
				startTile[0], endTile[0] = endTile[0], startTile[0]
			}
			if startTile[1] > endTile[1] {
				startTile[1], endTile[1] = endTile[1], startTile[1]
			}
			log.Printf("[make_geoimage] adding tiles from %v to %v", startTile, endTile)
			var tasks []skyhook.ExecTask
			for i := startTile[0]; i <= endTile[0]; i++ {
				for j := startTile[1]; j <= endTile[1]; j++ {
					tasks = append(tasks, skyhook.ExecTask{
						Key: fmt.Sprintf("%d_%d_%d", params.Zoom, i, j),
						Metadata: string(skyhook.JsonMarshal(TaskMetadata{i, j})),
					})
				}
			}
			return tasks, nil
		},
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
