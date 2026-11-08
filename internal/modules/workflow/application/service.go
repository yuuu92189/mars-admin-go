package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"mars-admin-go/internal/modules/workflow/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DefinitionRepository interface {
	CreateDefinition(ctx context.Context, definition domain.Definition) (domain.Definition, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (domain.Definition, error)
	ListDefinitions(ctx context.Context, status string, page, pageSize int) ([]domain.Definition, int64, error)
	PublishDefinition(ctx context.Context, id uuid.UUID) error
}

type RuntimeRepository interface {
	Start(ctx context.Context, definition domain.Definition, actorID uuid.UUID, input domain.StartInput, node domain.Node) (domain.Instance, domain.Task, error)
	GetInstance(ctx context.Context, id uuid.UUID, actor domain.Actor) (domain.Instance, error)
	ListInstances(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Instance, int64, error)
	Todo(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error)
	Done(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error)
	TaskContext(ctx context.Context, taskID uuid.UUID, actor domain.Actor) (domain.Task, domain.Instance, domain.Definition, domain.Node, error)
	Decide(ctx context.Context, actor domain.Actor, input domain.DecisionInput, decision domain.DOMDecision) (domain.Task, domain.Instance, error)
	Terminate(ctx context.Context, instanceID uuid.UUID, actor domain.Actor, comment string) (domain.Instance, error)
	Events(ctx context.Context, instanceID uuid.UUID, actor domain.Actor) ([]domain.Event, error)
}

type Repository interface {
	DefinitionRepository
	RuntimeRepository
}

type Service struct {
	definitions DefinitionRepository
	runtime     RuntimeRepository
}

func New(repo Repository) *Service {
	return &Service{definitions: repo, runtime: repo}
}

func (s *Service) CreateDefinition(ctx context.Context, createdBy uuid.UUID, input domain.CreateDefinitionInput) (domain.Definition, error) {
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || len(input.Code) > 100 {
		return domain.Definition{}, apperror.Validation("code", "Workflow code is required and must not exceed 100 characters.")
	}
	if input.Name == "" || len(input.Name) > 150 {
		return domain.Definition{}, apperror.Validation("name", "Workflow name is required and must not exceed 150 characters.")
	}
	if input.Version < 1 {
		input.Version = 1
	}
	if err := domain.ValidateDefinition(input.Document); err != nil {
		return domain.Definition{}, apperror.Validation("definition", err.Error())
	}
	definition := domain.Definition{
		ID: mustID(), Code: input.Code, Name: input.Name, Version: input.Version,
		Description: input.Description, Document: input.Document, Status: "draft",
		CreatedAt: time.Now().UTC(), CreatedBy: domain.CloneUUID(createdBy),
	}
	return s.definitions.CreateDefinition(ctx, definition)
}

func (s *Service) Definition(ctx context.Context, id uuid.UUID) (domain.Definition, error) {
	definition, err := s.definitions.GetDefinition(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Definition{}, apperror.NotFound("Workflow definition")
	}
	return definition, err
}

func (s *Service) Definitions(ctx context.Context, status string, page, pageSize int) ([]domain.Definition, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.definitions.ListDefinitions(ctx, status, page, pageSize)
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) error {
	definition, err := s.Definition(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.ValidateDefinition(definition.Document); err != nil {
		return apperror.Validation("definition", err.Error())
	}
	if err := s.definitions.PublishDefinition(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.Conflict("Only a draft workflow can be published.")
		}
		return err
	}
	return nil
}

func (s *Service) Start(ctx context.Context, actorID uuid.UUID, input domain.StartInput) (domain.Instance, domain.Task, error) {
	if input.BusinessID == "" || len(input.BusinessID) > 120 {
		return domain.Instance{}, domain.Task{}, apperror.Validation("businessId", "businessId is required and must not exceed 120 characters.")
	}
	definition, err := s.Definition(ctx, input.DefinitionID)
	if err != nil {
		return domain.Instance{}, domain.Task{}, err
	}
	if definition.Status != "published" {
		return domain.Instance{}, domain.Task{}, apperror.Conflict("Only a published workflow can be started.")
	}
	first, err := domain.FirstNode(definition.Document)
	if err != nil {
		return domain.Instance{}, domain.Task{}, apperror.Validation("definition", err.Error())
	}
	if input.Variables == nil {
		input.Variables = map[string]any{}
	}
	return s.runtime.Start(ctx, definition, actorID, input, first)
}

func (s *Service) Instance(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Instance, error) {
	instance, err := s.runtime.GetInstance(ctx, id, normalizeActor(actor))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Instance{}, apperror.NotFound("Workflow instance")
	}
	return instance, err
}

func (s *Service) Instances(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Instance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.runtime.ListInstances(ctx, normalizeActor(actor), page, pageSize)
}

func (s *Service) Todo(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.runtime.Todo(ctx, actor, page, pageSize)
}

func (s *Service) Done(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.runtime.Done(ctx, actor, page, pageSize)
}

func (s *Service) Decide(ctx context.Context, actor domain.Actor, input domain.DecisionInput) (domain.Task, domain.Instance, error) {
	if input.Action != "approve" && input.Action != "reject" {
		return domain.Task{}, domain.Instance{}, apperror.Validation("action", "action must be approve or reject.")
	}
	if len(input.Comment) > 2000 {
		return domain.Task{}, domain.Instance{}, apperror.Validation("comment", "Comment must not exceed 2000 characters.")
	}
	_, instance, definition, node, err := s.runtime.TaskContext(ctx, input.TaskID, actor)
	if err != nil {
		return domain.Task{}, domain.Instance{}, mapDecisionError(err)
	}
	variables := instance.Variables
	for key, value := range input.Variables {
		variables[key] = value
	}
	decision := domain.DOMDecision{Action: input.Action, Comment: input.Comment, Variables: variables}
	if input.Action == "reject" {
		decision.TerminalStatus = "rejected"
	} else {
		next, err := domain.NextNode(definition.Document, node, variables)
		if err != nil {
			return domain.Task{}, domain.Instance{}, apperror.Validation("variables", err.Error())
		}
		decision.NextNode = next
		if next == nil || next.Type == "end" {
			decision.TerminalStatus = "approved"
		}
	}
	task, updatedInstance, err := s.runtime.Decide(ctx, actor, input, decision)
	if err != nil {
		return domain.Task{}, domain.Instance{}, mapDecisionError(err)
	}
	return task, updatedInstance, nil
}

func (s *Service) Terminate(ctx context.Context, actor domain.Actor, instanceID uuid.UUID, comment string) (domain.Instance, error) {
	if len(comment) > 2000 {
		return domain.Instance{}, apperror.Validation("comment", "Comment must not exceed 2000 characters.")
	}
	normalized := normalizeActor(actor)
	if _, err := s.runtime.GetInstance(ctx, instanceID, normalized); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Instance{}, apperror.NotFound("Workflow instance")
		}
		return domain.Instance{}, err
	}
	instance, err := s.runtime.Terminate(ctx, instanceID, normalized, comment)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Instance{}, apperror.Conflict("Only a running workflow instance can be terminated.")
	}
	return instance, err
}

func (s *Service) Events(ctx context.Context, actor domain.Actor, instanceID uuid.UUID) ([]domain.Event, error) {
	events, err := s.runtime.Events(ctx, instanceID, normalizeActor(actor))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("Workflow instance")
	}
	return events, err
}

func normalizeActor(actor domain.Actor) domain.Actor {
	if actor.DataScope == "" {
		actor.DataScope = "self"
	}
	return actor
}

func mapDecisionError(err error) error {
	switch {
	case errors.Is(err, domain.ErrDoubleDecision):
		return apperror.Conflict("This task has already been completed.")
	case errors.Is(err, domain.ErrNotAssignee):
		return apperror.Forbidden("You are not an assignee for this task.")
	case errors.Is(err, pgx.ErrNoRows):
		return apperror.NotFound("Workflow task")
	default:
		return err
	}
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
