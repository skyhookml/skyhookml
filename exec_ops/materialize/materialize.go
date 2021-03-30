package virtual_debug

// An identity function, but materializes its input datasets.
// If inputs have non-default provider, then we copy them into a new dataset.
// (If they have default provider, we still copy them.)
//

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"fmt"
)

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "materialize",
			Name: "Materialize",
			Description: "Materialize input datasets",
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
					inItem := itemList[0]
					dataset := node.OutputDatasets[fmt.Sprintf("outputs%d", i)]
					item, err := exec_ops.AddItem(url, dataset, task.Key, inItem.Ext, inItem.Format, inItem.Metadata)
					if err != nil {
						return err
					}
					err = inItem.CopyTo(item.Fname(), inItem.Format, false)
					if err != nil {
						return err
					}
				}
				return nil
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		Incremental: true,
		GetOutputKeys: exec_ops.MapGetOutputKeys,
		GetNeededInputs: exec_ops.MapGetNeededInputs,
		ImageName: "skyhookml/basic",
	})
}