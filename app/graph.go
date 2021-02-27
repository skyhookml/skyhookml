package app

import (
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
	for name, plist := range node.GetParents() {
		for i, parent := range plist {
			if parent.Type == "n" {
				k := fmt.Sprintf("%s-%d-n[%s]", name, i, parent.Name)
				parents[k] = GetExecNode(parent.ID)
			} else if parent.Type == "d" {
				k := fmt.Sprintf("%s-%d-d", name, i)
				parents[k] = GetDataset(parent.ID)
			}
		}
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
	datasets, ok := node.GetDatasets(false)
	if !ok {
		return false
	}
	for _, ds := range datasets {
		if !ds.Done {
			return false
		}
	}
	return true
}

func (node *DBExecNode) GraphID() string {
	return fmt.Sprintf("exec-%d", node.ID)
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
