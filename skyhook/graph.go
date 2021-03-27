package skyhook

// Defines types that represent execution pipeline graphs.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// A node in an execution pipeline.
// This is either an execution node (VirtualNode) or a dataset (Dataset).
type Node interface {
	// Parents of this node.
	// These are other nodes that must be executed before this node can be executed.
	// Key defines what kind of parent it is.
	GraphParents() map[string]GraphID
	// The local hash at this node.
	// The hash of a node is computed by merging hash of its parents with its local hash.
	LocalHash() []byte
	// unique identifier
	GraphID() GraphID
}

// ID type in execution graph.
type GraphID struct {
	// Either "exec" or "dataset"
	Type string
	// ID of ExecNode or Dataset
	ID int
	// For virtual nodes, some unique key for it
	VirtualKey string
}

// An execption graph, that maps GraphIDs to Nodes.
type ExecutionGraph map[GraphID]Node

// Returns hashes of all nodes in the graph.
func (graph ExecutionGraph) GetHashes() map[GraphID][]byte {
	hashes := make(map[GraphID][]byte)
	// repeatedly iterate over graph and add hashes for nodes
	// where parent hashes are already computed
	for len(hashes) < len(graph) {
		for graphID, node := range graph {
			if hashes[graphID] != nil {
				continue
			}
			// collect parent hashes
			missing := false
			parentHashes := make(map[string][]byte)
			var parentKeys []string
			for k, parentID := range node.GraphParents() {
				if parentID.VirtualKey != "" {
					// Hash only considers non-virtual nodes.
					// Because virtual ones are not known statically.
					continue
				}
				if hashes[parentID] == nil {
					missing = true
					break
				}
				parentKeys = append(parentKeys, k)
				parentHashes[k] = hashes[parentID]
			}
			if missing {
				continue
			}

			h := sha256.New()
			sort.Strings(parentKeys)
			for _, k := range parentKeys {
				hash := parentHashes[k]
				h.Write([]byte(fmt.Sprintf("%s=%s\n", k, string(hash))))
			}
			h.Write(node.LocalHash())
			hashes[graphID] = h.Sum(nil)
		}
	}
	return hashes
}

func (graph ExecutionGraph) GetHashStrings() map[GraphID]string {
	hashes := make(map[GraphID]string)
	for graphID, bytes := range graph.GetHashes() {
		hashes[graphID] = hex.EncodeToString(bytes)
	}
	return hashes
}

// Like ExecParent but points to an actual Node.
type VirtualParent struct {
	GraphID GraphID
	// if GraphID.Type is "exec", then this is name of output that we want to input
	Name string

	DataType DataType
}

// Like ExecNode, but knows its position in the dynamic execution graph.
// So it points to parent nodes/datasets with GraphIDs instead of database IDs.
// This is virtual because it may or may not be an actual configured node.
// (It may by created dynamically by ExecOpImpl.Resolve.)
type VirtualNode struct {
	// from ExecNode
	Name string
	Op string
	Params string

	// Parents of this node.
	Parents map[string][]VirtualParent
	// the concrete node that this ExecNode was created from
	// if ExecOpImpl.Resolve is not set, then Node == OrigNode
	OrigNode ExecNode
	// A unique identifier for this node if it is actually virtual (Node != OrigNode).
	VirtualKey string
}

func (node VirtualNode) GetOp() ExecOpProvider {
	return GetExecOp(node.Op)
}

func (node VirtualNode) GetInputs() []ExecInput {
	return node.GetOp().GetInputs(node.Params)
}

func (node VirtualNode) GetInputTypes() map[string][]DataType {
	inputTypes := make(map[string][]DataType)
	for _, input := range node.GetInputs() {
		for _, parent := range node.Parents[input.Name] {
			inputTypes[input.Name] = append(inputTypes[input.Name], parent.DataType)
		}
	}
	return inputTypes
}

func (node VirtualNode) GetOutputs() []ExecOutput {
	return node.GetOp().GetOutputs(node.Params, node.GetInputTypes())
}

func (node VirtualNode) GraphParents() map[string]GraphID {
	parents := make(map[string]GraphID)
	for name, plist := range node.Parents {
		for i, parent := range plist {
			if parent.GraphID.Type == "exec" {
				k := fmt.Sprintf("%s-%d-n[%s]", name, i, parent.Name)
				parents[k] = parent.GraphID
			} else if parent.GraphID.Type == "dataset" {
				k := fmt.Sprintf("%s-%d-d", name, i)
				parents[k] = parent.GraphID
			}
		}
	}
	return parents
}

func (node VirtualNode) GraphID() GraphID {
	return GraphID{
		Type: "exec",
		ID: node.OrigNode.ID,
		VirtualKey: node.VirtualKey,
	}
}

func (node VirtualNode) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("op=%s\n", node.Op)))
	h.Write([]byte(fmt.Sprintf("params=%s\n", node.Params)))
	return h.Sum(nil)
}

func (node VirtualNode) GetRunnable(inputDatasets map[string][]Dataset, outputDatasets map[string]Dataset) Runnable {
	return Runnable{
		Name: node.Name,
		Op: node.Op,
		Params: node.Params,
		InputDatasets: inputDatasets,
		OutputDatasets: outputDatasets,
	}
}

func (node Dataset) GraphParents() map[string]GraphID {
	return nil
}
func (node Dataset) GraphID() GraphID {
	return GraphID{
		Type: "dataset",
		ID: node.ID,
	}
}
