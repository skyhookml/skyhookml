package virtual_debug

// Merge all input items into one output item.
// For table inputs, this is like SQL UNION operation.

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"io"
)

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "union",
			Name: "Union",
			Description: "Combine many items into one",
		},
		Inputs: []skyhook.ExecInput{{Name: "input"}},
		GetOutputs: func(params string, inputTypes map[string][]skyhook.DataType) []skyhook.ExecOutput {
			if len(inputTypes["input"]) == 0 {
				return nil
			}
			return []skyhook.ExecOutput{{Name: "output", DataType: inputTypes["input"][0]}}
		},
		Requirements: func(node skyhook.Runnable) map[string]int {
			return nil
		},
		GetTasks: exec_ops.SingleTask("union"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				items := task.Items["input"][0]
				outDataset := node.OutputDatasets["output"]
				dtype := outDataset.DataType
				// we support:
				// - Table: combine all rows into one table.
				// - Any sequence type: use ChunkBuilder to build the sequence.
				if dtype == skyhook.TableType {
					outTable := skyhook.TableData{}
					for _, item := range items {
						data_, err := item.LoadData()
						if err != nil {
							return err
						}
						data := data_.(skyhook.TableData)
						if outTable.Specs == nil {
							outTable.Specs = data.Specs
						}
						outTable.Data = append(outTable.Data, data.Data...)
					}
					return exec_ops.WriteItem(url, outDataset, "union", outTable)
				} else {
					// must be sequence data
					builder := skyhook.DataImpls[dtype].Builder()
					for _, item := range items {
						data, err := item.LoadData()
						if err != nil {
							return err
						}
						rd := data.(skyhook.ReadableData).Reader()
						for {
							cur, err := rd.Read(32)
							if err == io.EOF {
								break
							} else if err != nil {
								return err
							}
							builder.Write(cur)
						}
					}
					data, err := builder.Close()
					if err != nil {
						return err
					}
					return exec_ops.WriteItem(url, outDataset, "union", data)
				}
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
