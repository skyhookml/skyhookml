package skyhook

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
)

type ExecParent struct {
	// "n" for ExecNode, "d" for Dataset
	Type string
	ID int

	// index of ExecNode output that is being input
	Index int
}

func (p ExecParent) String() string {
	var parts []string
	parts = append(parts, p.Type)
	parts = append(parts, strconv.Itoa(p.ID))
	if p.Type == "n" {
		parts = append(parts, strconv.Itoa(p.Index))
	}
	return strings.Join(parts, ",")
}

func ExecParentsToString(parents []ExecParent) string {
	var strs []string
	for _, parent := range parents {
		strs = append(strs, parent.String())
	}
	return strings.Join(strs, ";")
}

func ParseExecParent(s string) ExecParent {
	parts := strings.Split(s, ",")
	p := ExecParent{
		Type: parts[0],
		ID: ParseInt(parts[1]),
	}
	if p.Type == "n" {
		p.Index = ParseInt(parts[2])
	}
	return p
}

func ParseExecParents(s string) []ExecParent {
	if s == "" {
		return []ExecParent{}
	}
	parts := strings.Split(s, ";")
	parents := make([]ExecParent, len(parts))
	for i, part := range parts {
		parents[i] = ParseExecParent(part)
	}
	return parents
}

type ExecNode struct {
	ID int
	Name string
	Op string
	Params string
	Parents []ExecParent
	DataTypes []DataType
}

func (node ExecNode) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("op=%s\n", node.Op)))
	h.Write([]byte(fmt.Sprintf("params=%s\n", node.Params)))
	h.Write([]byte(fmt.Sprintf("datatypes=%s\n", EncodeTypes(node.DataTypes))))
	return h.Sum(nil)
}
