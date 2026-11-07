package domain

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func ValidateDefinition(document DefinitionDocument) error {
	if strings.TrimSpace(document.Start) == "" {
		return fmt.Errorf("%w: start node is required", ErrInvalidDefinition)
	}
	if len(document.Nodes) == 0 || len(document.Nodes) > 200 {
		return fmt.Errorf("%w: workflow must contain 1 to 200 nodes", ErrInvalidDefinition)
	}
	nodes := make(map[string]Node, len(document.Nodes))
	for _, node := range document.Nodes {
		node.Code = strings.TrimSpace(node.Code)
		if node.Code == "" || len(node.Code) > 100 {
			return fmt.Errorf("%w: node code is required and must not exceed 100 characters", ErrInvalidDefinition)
		}
		if _, exists := nodes[node.Code]; exists {
			return fmt.Errorf("%w: duplicate node %q", ErrInvalidDefinition, node.Code)
		}
		if node.Type != "approval" && node.Type != "end" {
			return fmt.Errorf("%w: node %q has unsupported type", ErrInvalidDefinition, node.Code)
		}
		if node.Type == "approval" && node.AssigneeID == nil && node.AssigneeRole == "" {
			return fmt.Errorf("%w: approval node %q requires an assignee", ErrInvalidDefinition, node.Code)
		}
		nodes[node.Code] = node
	}
	if _, exists := nodes[document.Start]; !exists {
		return fmt.Errorf("%w: start node does not exist", ErrInvalidDefinition)
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) error
	visit = func(code string) error {
		if visiting[code] {
			return fmt.Errorf("%w: cycle detected at node %q", ErrInvalidDefinition, code)
		}
		if visited[code] {
			return nil
		}
		visiting[code] = true
		node := nodes[code]
		nextCodes := make([]string, 0)
		if node.Next != "" {
			nextCodes = append(nextCodes, node.Next)
		}
		for _, route := range node.Routes {
			nextCodes = append(nextCodes, route.Next)
			if strings.TrimSpace(route.Condition) == "" {
				return fmt.Errorf("%w: route condition is required", ErrInvalidDefinition)
			}
			if _, err := expr.Compile(route.Condition, expr.AllowUndefinedVariables()); err != nil {
				return fmt.Errorf("%w: invalid route condition: %w", ErrInvalidDefinition, err)
			}
		}
		for _, next := range nextCodes {
			if _, ok := nodes[next]; !ok {
				return fmt.Errorf("%w: node %q references missing node %q", ErrInvalidDefinition, code, next)
			}
			if err := visit(next); err != nil {
				return err
			}
		}
		visiting[code] = false
		visited[code] = true
		return nil
	}
	return visit(document.Start)
}

func FirstNode(document DefinitionDocument) (Node, error) {
	for _, node := range document.Nodes {
		if node.Code == document.Start {
			return node, nil
		}
	}
	return Node{}, ErrInvalidDefinition
}

func NextNode(document DefinitionDocument, current Node, variables map[string]any) (*Node, error) {
	nextCode := current.Next
	for _, route := range current.Routes {
		program, err := expr.Compile(route.Condition, expr.AllowUndefinedVariables())
		if err != nil {
			return nil, err
		}
		result, err := expr.Run(program, variables)
		if err != nil {
			return nil, err
		}
		if matched, _ := result.(bool); matched {
			nextCode = route.Next
			break
		}
	}
	if nextCode == "" {
		return nil, nil
	}
	for _, node := range document.Nodes {
		if node.Code == nextCode {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("workflow route references missing node %q", nextCode)
}

func IsAssignee(node Node, actor Actor) bool {
	if node.AssigneeID != nil && *node.AssigneeID == actor.UserID {
		return true
	}
	if node.AssigneeRole != "" {
		for _, role := range actor.Roles {
			if role == node.AssigneeRole {
				return true
			}
		}
	}
	return false
}

func CloneUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	value := id
	return &value
}
