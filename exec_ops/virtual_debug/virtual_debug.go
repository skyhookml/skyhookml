package virtual_debug

// An op for debugging that applies an identity function.
// It wraps items in the input datasets under a virtual provider that removes
// the filename reference. So it's useful for testing to make sure that all ops
// are properly handling cases where Item.Fname is not available.

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
	"log"
	urllib "net/url"
)

func init() {
	skyhook.ItemProviders["virtual_debug"] = skyhook.VirtualProvider(func(item skyhook.Item, data skyhook.Data) (skyhook.Data, error) {
		return data, nil
	}, false)

	skyhook.ExecOpImpls["virtual_debug"] = skyhook.ExecOpImpl{
		Requirements: func(url string, node skyhook.ExecNode) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SimpleTasks,
		Prepare: func(url string, node skyhook.ExecNode, outputDatasets map[string]skyhook.Dataset) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				for i, itemList := range task.Items["inputs"] {
					item := itemList[0]
					dataset := outputDatasets[fmt.Sprintf("outputs%d", i)]
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
		GetOutputs: func(url string, node skyhook.ExecNode) []skyhook.ExecOutput {
			// output outputs0, outputs1, ... for each dataset in inputs

			// return empty string on error
			getOutputType := func(parent skyhook.ExecParent) skyhook.DataType {
				dataType, err := exec_ops.ParentToDataType(url, parent)
				if err != nil {
					log.Printf("[render] warning: unable to compute outputs: %v", err)
					return ""
				}
				return dataType
			}

			parents := node.GetParents()
			var outputs []skyhook.ExecOutput
			for i, parent := range parents["inputs"] {
				dataType := getOutputType(parent)
				if dataType == "" {
					return node.Outputs
				}
				outputs = append(outputs, skyhook.ExecOutput{
					Name: fmt.Sprintf("outputs%d", i),
					DataType: dataType,
				})
			}
			return outputs
		},
		ImageName: func(url string, node skyhook.ExecNode) (string, error) {
			return "skyhookml/basic", nil
		},
	}
}
