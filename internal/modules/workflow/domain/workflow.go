package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type DefinitionDocument struct {
	Start string         `json:"start"`
	Nodes []Node         `json:"nodes"`
	Meta  map[string]any `json:"meta,omitempty"`
}

type Node struct {
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	AssigneeID   *uuid.UUID `json:"assigneeId,omitempty"`
	AssigneeRole string     `json:"assigneeRole,omitempty"`
	Next         string     `json:"next,omitempty"`
	Routes       []Route    `json:"routes,omitempty"`
	DueHours     int32      `json:"dueHours,omitempty"`
}

type Route struct {
	Condition string `json:"condition"`
	Next      string `json:"next"`
}

type Definition struct {
	ID          uuid.UUID          `json:"id"`
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Version     int32              `json:"version"`
	Description string             `json:"description"`
	Document    DefinitionDocument `json:"definition"`
	Status      string             `json:"status"`
	CreatedBy   *uuid.UUID         `json:"createdBy,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	PublishedAt *time.Time         `json:"publishedAt,omitempty"`
}

type CreateDefinitionInput struct {
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Version     int32              `json:"version"`
	Description string             `json:"description"`
	Document    DefinitionDocument `json:"definition"`
}

type Instance struct {
	ID              uuid.UUID      `json:"id"`
	DefinitionID    uuid.UUID      `json:"definitionId"`
	DefinitionName  string         `json:"definitionName,omitempty"`
	BusinessID      string         `json:"businessId"`
	InitiatorID     uuid.UUID      `json:"initiatorId"`
	Status          string         `json:"status"`
	CurrentNodeCode string         `json:"currentNodeCode,omitempty"`
	Variables       map[string]any `json:"variables"`
	StartedAt       time.Time      `json:"startedAt"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty"`
}

type Task struct {
	ID             uuid.UUID  `json:"id"`
	InstanceID     uuid.UUID  `json:"instanceId"`
	DefinitionID   uuid.UUID  `json:"definitionId,omitempty"`
	DefinitionName string     `json:"definitionName,omitempty"`
	BusinessID     string     `json:"businessId,omitempty"`
	NodeCode       string     `json:"nodeCode"`
	NodeName       string     `json:"nodeName"`
	AssigneeID     *uuid.UUID `json:"assigneeId,omitempty"`
	AssigneeRole   string     `json:"assigneeRole,omitempty"`
	Status         string     `json:"status"`
	Decision       string     `json:"decision,omitempty"`
	Comment        string     `json:"comment,omitempty"`
	DueAt          *time.Time `json:"dueAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
}

type Event struct {
	ID          uuid.UUID      `json:"id"`
	InstanceID  uuid.UUID      `json:"instanceId"`
	TaskID      *uuid.UUID     `json:"taskId,omitempty"`
	ActorUserID *uuid.UUID     `json:"actorUserId,omitempty"`
	Type        string         `json:"type"`
	FromNode    string         `json:"fromNode,omitempty"`
	ToNode      string         `json:"toNode,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type StartInput struct {
	DefinitionID uuid.UUID      `json:"definitionId"`
	BusinessID   string         `json:"businessId"`
	Variables    map[string]any `json:"variables"`
}

type DecisionInput struct {
	TaskID    uuid.UUID      `json:"taskId"`
	Action    string         `json:"action"`
	Comment   string         `json:"comment"`
	Variables map[string]any `json:"variables"`
}

type Actor struct {
	UserID uuid.UUID
	Roles  []string
}

type Repository interface {
	CreateDefinition(ctx context.Context, definition Definition) (Definition, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (Definition, error)
	ListDefinitions(ctx context.Context, status string, page, pageSize int) ([]Definition, int64, error)
	PublishDefinition(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, definition Definition, actorID uuid.UUID, input StartInput, node Node) (Instance, Task, error)
	GetInstance(ctx context.Context, id uuid.UUID) (Instance, error)
	ListInstances(ctx context.Context, page, pageSize int) ([]Instance, int64, error)
	Todo(ctx context.Context, actor Actor, page, pageSize int) ([]Task, int64, error)
	Done(ctx context.Context, actor Actor, page, pageSize int) ([]Task, int64, error)
	TaskContext(ctx context.Context, taskID uuid.UUID, actor Actor) (Task, Instance, Definition, Node, error)
	Decide(ctx context.Context, actor Actor, input DecisionInput, decision DOMDecision) (Task, Instance, error)
	Terminate(ctx context.Context, instanceID, actorID uuid.UUID, comment string) (Instance, error)
	Events(ctx context.Context, instanceID uuid.UUID) ([]Event, error)
}

type DOMDecision struct {
	Action         string
	Comment        string
	Variables      map[string]any
	NextNode       *Node
	TerminalStatus string
}

var (
	ErrInvalidDefinition = errors.New("invalid workflow definition")
	ErrDoubleDecision    = errors.New("workflow task has already been completed")
	ErrNotAssignee       = errors.New("actor is not an assignee for this task")
)
