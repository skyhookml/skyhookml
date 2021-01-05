package app

import (
	"../skyhook"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

type Node interface {
	GraphParents() map[string]Node
	LocalHash() []byte
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
	for i, p := range node.FilterParents {
		if p.Type == "n" {
			k := fmt.Sprintf("filter%d-n[%d]", i, p.Index)
			parents[k] = GetExecNode(p.ID)
		} else if p.Type == "d" {
			k := fmt.Sprintf("filter%d-d", i)
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

func (node *DBDataset) GraphParents() map[string]Node {
	return nil
}
