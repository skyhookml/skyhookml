package skyhook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ExecParent struct {
	// "n" for ExecNode, "d" for Dataset
	Type string
	ID int

	// name of ExecNode output that is being input
	Name string
}

func (p ExecParent) String() string {
	var parts []string
	parts = append(parts, p.Type)
	parts = append(parts, strconv.Itoa(p.ID))
	if p.Type == "n" {
		parts = append(parts, p.Name)
	}
	return strings.Join(parts, ",")
}

func ExecParentsToString(parents [][]ExecParent) string {
	var iparts []string
	for _, plist := range parents {
		var jparts []string
		for _, parent := range plist {
			jparts = append(jparts, parent.String())
		}
		iparts = append(iparts, strings.Join(jparts, ";"))
	}
	return strings.Join(iparts, "|")
}

func ParseExecParent(s string) ExecParent {
	parts := strings.Split(s, ",")
	p := ExecParent{
		Type: parts[0],
		ID: ParseInt(parts[1]),
	}
	if p.Type == "n" {
		p.Name = strings.Join(parts[2:], ",")
	}
	return p
}

func ParseExecParents(s string) [][]ExecParent {
	if s == "" {
		return [][]ExecParent{}
	}
	iparts := strings.Split(s, "|")
	parents := make([][]ExecParent, len(iparts))
	for i, ipart := range iparts {
		if ipart == "" {
			parents[i] = []ExecParent{}
			continue
		}
		jparts := strings.Split(ipart, ";")
		parents[i] = make([]ExecParent, len(jparts))
		for j, jpart := range jparts {
			parents[i][j] = ParseExecParent(jpart)
		}
	}
	return parents
}

type ExecInput struct {
	Name string
	// nil if input can be any type
	DataTypes []DataType
	// true if this node can accept multiple inputs for this name
	Variable bool
}

func ExecInputsToString(inputs []ExecInput) string {
	return string(JsonMarshal(inputs))
}

func ParseExecInputs(s string) []ExecInput {
	var inputs []ExecInput
	err := json.Unmarshal([]byte(s), &inputs)
	if err == nil {
		return inputs
	} else {
		return []ExecInput{}
	}
}

type ExecOutput struct {
	Name string
	DataType DataType
}

func ExecOutputsToString(outputs []ExecOutput) string {
	return string(JsonMarshal(outputs))
}

func ParseExecOutputs(s string) []ExecOutput {
	var outputs []ExecOutput
	err := json.Unmarshal([]byte(s), &outputs)
	if err == nil {
		return outputs
	} else {
		return []ExecOutput{}
	}
}

type ExecNode struct {
	ID int
	Name string
	Op string
	Params string

	// specify the expected inputs and structure of the outputs
	Inputs []ExecInput
	Outputs []ExecOutput

	// currently configured parents for each input
	// len(Parents) should be same as len(Inputs) (unless Parents is nil)
	Parents [][]ExecParent
}

func (node ExecNode) LocalHash() []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("op=%s\n", node.Op)))
	h.Write([]byte(fmt.Sprintf("params=%s\n", node.Params)))
	h.Write([]byte(fmt.Sprintf("outputs=%s\n", ExecOutputsToString(node.Outputs))))
	return h.Sum(nil)
}

// Transforms Parents into a map from the input name.
func (node ExecNode) GetParents() map[string][]ExecParent {
	parents := make(map[string][]ExecParent)
	for i := 0; i < len(node.Inputs) && i < len(node.Parents); i++ {
		input := node.Inputs[i]
		for _, parent := range node.Parents[i] {
			parents[input.Name] = append(parents[input.Name], parent)
		}
	}
	return parents
}
