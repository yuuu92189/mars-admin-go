package domain

import "testing"

func FuzzValidateDefinition(f *testing.F) {
	f.Add("", "")
	f.Add("start", "approval")
	f.Add("a", "end")
	f.Fuzz(func(t *testing.T, code, nodeType string) {
		document := DefinitionDocument{
			Start: code,
			Nodes: []Node{{Code: code, Name: "node", Type: nodeType}},
		}
		_ = ValidateDefinition(document)
	})
}
