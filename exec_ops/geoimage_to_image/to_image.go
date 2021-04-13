package geoimage_to_image

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	urllib "net/url"
)

const MyName string = "geoimage_to_image"

func init() {
	skyhook.ItemProviders[MyName] = skyhook.VirtualProvider(func(item skyhook.Item, data skyhook.Data) (skyhook.Data, error) {
		im, err := data.(skyhook.GeoImageData).GetImage()
		if err != nil {
			return nil, err
		}
		return skyhook.ImageData{Images: []skyhook.Image{im}}, nil
	}, false)

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: MyName,
			Name: "Geo-Image to Image",
			Description: "Use a Geo-Image dataset as an Image type",
		},
		Inputs: []skyhook.ExecInput{{Name: "input", DataTypes: []skyhook.DataType{skyhook.GeoImageType}}},
		Outputs: []skyhook.ExecOutput{{Name: "output", DataType: skyhook.ImageType}},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				item := task.Items["input"][0][0]
				dataset := node.OutputDatasets["output"]
				return skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
					"key": {task.Key},
					"ext": {"jpg"},
					"format": {"jpeg"},
					"metadata": {""},
					"provider": {MyName},
					"provider_info": {string(skyhook.JsonMarshal(item))},
				}, nil)
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
