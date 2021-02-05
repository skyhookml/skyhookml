package app

import (
	"../skyhook"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
)

type Node interface {
	GraphParents() map[string]Node
	LocalHash() []byte
	GraphType() string
	IsDone() bool

	// return unique ID for this node
	GraphID() string
}

func GetNodeHash(node Node) []byte {
	parentHashes := make(map[string][]byte)
	var parentKeys []string
	for k, n := range node.GraphParents() {
		parentKeys = append(parentKeys, k)
		parentHashes[k] = GetNodeHash(n)
	}

	h := sha256.New()
	sort.Strings(parentKeys)
	for _, k := range parentKeys {
		hash := parentHashes[k]
		h.Write([]byte(fmt.Sprintf("%s=%s\n", k, string(hash))))
	}
	h.Write(node.LocalHash())
	return h.Sum(nil)
}

func (node *DBExecNode) GraphParents() map[string]Node {
	parents := make(map[string]Node)
	for i, p := range node.Parents {
		if p.Type == "n" {
			k := fmt.Sprintf("%d-n[%d]", i, p.Index)
			parents[k] = GetExecNode(p.ID)
		} else if p.Type == "d" {
			k := fmt.Sprintf("%d-d", i)
			parents[k] = GetDataset(p.ID)
		}
	}

	if node.Op == "model" {
		var params skyhook.ModelExecParams
		skyhook.JsonUnmarshal([]byte(node.Params), &params)
		parents["model"] = GetTrainNode(params.TrainNodeID)
	}

	return parents
}

func (node *DBExecNode) Hash() string {
	bytes := GetNodeHash(node)
	return hex.EncodeToString(bytes)
}

func (node *DBExecNode) GraphType() string {
	return "exec"
}

func (node *DBExecNode) IsDone() bool {
	_, ok := node.GetDatasets(false)
	// TODO: make sure no datasets are only partially computed
	return ok
}

func (node *DBExecNode) GraphID() string {
	return fmt.Sprintf("exec-%d", node.ID)
}

func (node *DBTrainNode) GraphParents() map[string]Node {
	parents := make(map[string]Node)
	for i, id := range node.ParentIDs {
		parents[fmt.Sprintf("%d", i)] = GetTrainNode(id)
	}
	return parents
}

func (node *DBTrainNode) Hash() string {
	bytes := GetNodeHash(node)
	return hex.EncodeToString(bytes)
}

func (node *DBTrainNode) GraphType() string {
	return "train"
}

func (node *DBTrainNode) IsDone() bool {
	if node.ModelID == nil {
		return false
	}
	model := GetModel(*node.ModelID)
	return model.Hash == node.Hash()
}

func (node *DBTrainNode) GraphID() string {
	return fmt.Sprintf("train-%d", node.ID)
}

func (node *DBDataset) GraphParents() map[string]Node {
	return nil
}
func (node *DBDataset) GraphType() string {
	return "dataset"
}
func (node *DBDataset) IsDone() bool {
	return true
}
func (node *DBDataset) GraphID() string {
	return fmt.Sprintf("dataset-%d", node.ID)
}

// Run the specified node, while running ancestors first if needed.
func RunTree(node Node) error {
	if node.IsDone() {
		return nil
	}

	needed := map[string]Node{node.GraphID(): node}
	q := []Node{node}
	for len(q) > 0 {
		cur := q[len(q)-1]
		q = q[0:len(q)-1]
		for _, parent := range cur.GraphParents() {
			if parent.IsDone() {
				continue
			}
			if needed[parent.GraphID()] != nil {
				continue
			}
			needed[parent.GraphID()] = parent
		}
	}

	log.Printf("[run-tree %v] running %d nodes", node, len(needed))
	for len(needed) > 0 {
		for id, cur := range needed {
			ready := true
			for _, parent := range cur.GraphParents() {
				if !parent.IsDone() {
					ready = false
					break
				}
			}
			if !ready {
				continue
			}

			var err error
			if cur.GraphType() == "exec" {
				err = cur.(*DBExecNode).Run(ExecRunOptions{})
			} else if cur.GraphType() == "train" {
				err = cur.(*DBTrainNode).Run(false)
			} else {
				err = fmt.Errorf("unknown type %s", cur.GraphType())
			}
			if err != nil {
				return err
			}
			delete(needed, id)
		}
	}

	return nil
}