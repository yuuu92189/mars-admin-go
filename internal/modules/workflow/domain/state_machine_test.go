package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateDefinitionRejectsCycle(t *testing.T) {
	role := "manager"
	doc := DefinitionDocument{
		Start: "a",
		Nodes: []Node{
			{Code: "a", Name: "A", Type: "approval", AssigneeRole: role, Next: "b"},
			{Code: "b", Name: "B", Type: "approval", AssigneeRole: role, Next: "a"},
		},
	}
	if err := ValidateDefinition(doc); err == nil {
		t.Fatal("expected cycle to be rejected")
	}
}

func TestNextNodeCondition(t *testing.T) {
	role := "manager"
	doc := DefinitionDocument{
		Start: "review",
		Nodes: []Node{
			{
				Code: "review", Name: "Review", Type: "approval", AssigneeRole: role,
				Routes: []Route{
					{Condition: "amount > 1000", Next: "director"},
					{Condition: "true", Next: "done"},
				},
			},
			{Code: "director", Name: "Director", Type: "approval", AssigneeRole: "director", Next: "done"},
			{Code: "done", Name: "Done", Type: "end"},
		},
	}
	if err := ValidateDefinition(doc); err != nil {
		t.Fatal(err)
	}
	next, err := NextNode(doc, doc.Nodes[0], map[string]any{"amount": float64(2500)})
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || next.Code != "director" {
		t.Fatalf("expected director node, got %#v", next)
	}
}

func TestIsAssigneeByRole(t *testing.T) {
	userID := uuid.New()
	node := Node{AssigneeRole: "manager"}
	if !IsAssignee(node, Actor{UserID: userID, Roles: []string{"manager"}}) {
		t.Fatal("expected role assignee")
	}
}
