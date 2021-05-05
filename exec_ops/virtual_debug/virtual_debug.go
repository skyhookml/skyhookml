package virtual_debug

// An op for debugging that applies an identity function.
// It wraps items in the input datasets under a virtual provider that removes
// the filename reference. So it's useful for testing to make sure that all ops
// are properly handling cases where Item.Fname is not available.

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	urllib "net/url"
)

func init() {
	skyhook.ItemProviders["virtual_debug"] = skyhook.VirtualProvider(func(item skyhook.Item, data interface{}, metadata skyhook.DataMetadata) (interface{}, skyhook.DataMetadata, error) {
		return data, metadata, nil
	}, false)

	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "virtual_debug",
			Name: "Virtual Debug",
			Description: "Op implementing identity function with a virtual provider",
		},
		Inputs: []skyhook.ExecInput{{Name: "inputs", Variable: true}},
		GetOutputs: exec_ops.GetOutputsSimilarToInputs,
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				for i, itemList := range task.Items["inputs"] {
					item := itemList[0]
					dataset := node.OutputDatasets[fmt.Sprintf("outputs%d", i)]
					err := skyhook.JsonPostForm(url, fmt.Sprintf("/datasets/%d/items", dataset.ID), urllib.Values{
						"key": {task.Key},
						"ext": {item.Ext},
						"format": {item.Format},
						"metadata": {item.Metadata},
						"provider": {"virtual_debug"},
						"provider_info": {string(skyhook.JsonMarshal(item))},
					}, nil)
					if err != nil {
						return err
					}
				}
				return nil
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
