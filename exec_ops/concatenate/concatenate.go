package concatenate

// Merge all input items into one output item.
// For table inputs, this is like SQL UNION operation, at least within one dataset.

import (
	"github.com/skyhookml/skyhookml/skyhook"
	"github.com/skyhookml/skyhookml/exec_ops"

	"io"
)

func init() {
	skyhook.AddExecOpImpl(skyhook.ExecOpImpl{
		Config: skyhook.ExecOpConfig{
			ID: "concatenate",
			Name: "Concatenate",
			Description: "Merge all items in the input dataset into one item in the output dataset",
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
		GetTasks: exec_ops.SingleTask("concatenate"),
		Prepare: func(url string, node skyhook.Runnable) (skyhook.ExecOp, error) {
			applyFunc := func(task skyhook.ExecTask) error {
				items := task.Items["input"][0]
				outDataset := node.OutputDatasets["output"]
				dtype := outDataset.DataType
				// we support:
				// - Table: combine all rows into one table.
				// - Any sequence type: use ChunkBuilder to build the sequence.
				// TODO: make Table a sequence type so we don't have to have this special case.
				if dtype == skyhook.TableType {
					var outData [][]string
					var outMetadata skyhook.DataMetadata
					for _, item := range items {
						data, metadata, err := item.LoadData()
						if err != nil {
							return err
						}
						outMetadata = metadata
						outData = append(outData, data.([][]string)...)
					}
					return exec_ops.WriteItem(url, outDataset, "concatenate", outData, outMetadata)
				} else {
					// Must be sequence data.
					// Get the metadata, ext, and format from the first item.
					outItem, err := exec_ops.AddItem(url, outDataset, "concatenate", items[0].Ext, items[0].Format, items[0].DecodeMetadata())
					if err != nil {
						return err
					}
					writer := outItem.LoadWriter()
					for _, item := range items {
						reader, _ := item.LoadReader()
						for {
							cur, err := reader.Read(32)
							if err == io.EOF {
								break
							} else if err != nil {
								return err
							}
							writer.Write(cur)
						}
						reader.Close()
					}
					return writer.Close()
				}
			}
			return skyhook.SimpleExecOp{ApplyFunc: applyFunc}, nil
		},
		ImageName: "skyhookml/basic",
	})
}
