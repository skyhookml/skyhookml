package geoimage_to_image

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	urllib "net/url"
)

const MyName string = "geoimage_to_image"

func init() {
	myProviderFunc := func(item skyhook.Item, data interface{}, metadata skyhook.DataMetadata) (interface{}, skyhook.DataMetadata, error) {
		return data, skyhook.NoMetadata{}, nil
	}
	skyhook.ItemProviders[MyName] = skyhook.VirtualProvider(myProviderFunc, false)

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
			var params struct {
				Materialize bool
			}
			if err := exec_ops.DecodeParams(node, &params, true); err != nil {
				return nil, err
			}
			applyFunc := func(task skyhook.ExecTask) error {
				item := task.Items["input"][0][0]
				dataset := node.OutputDatasets["output"]
				if params.Materialize {
					// A loaded GeoImage is just a skyhook.Image.
					// So we can directly write that to the Image dataset.
					data, _, err := item.LoadData()
					if err != nil {
						return err
					}
					return exec_ops.WriteItem(url, dataset, task.Key, data, skyhook.NoMetadata{})
				} else {
					return skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
						"key": {task.Key},
						"ext": {"jpg"},
						"format": {"jpeg"},
						"metadata": {""},
						"provider": {MyName},
						"provider_info": {string(skyhook.JsonMarshal(item))},
					}, nil)
				}
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
