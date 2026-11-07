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

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
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
	return s.repo.CreateDefinition(ctx, definition)
}

func (s *Service) Definition(ctx context.Context, id uuid.UUID) (domain.Definition, error) {
	definition, err := s.repo.GetDefinition(ctx, id)
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
	return s.repo.ListDefinitions(ctx, status, page, pageSize)
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) error {
	definition, err := s.Definition(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.ValidateDefinition(definition.Document); err != nil {
		return apperror.Validation("definition", err.Error())
	}
	if err := s.repo.PublishDefinition(ctx, id); err != nil {
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
	return s.repo.Start(ctx, definition, actorID, input, first)
}

func (s *Service) Instance(ctx context.Context, id uuid.UUID) (domain.Instance, error) {
	instance, err := s.repo.GetInstance(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Instance{}, apperror.NotFound("Workflow instance")
	}
	return instance, err
}

func (s *Service) Instances(ctx context.Context, page, pageSize int) ([]domain.Instance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListInstances(ctx, page, pageSize)
}

func (s *Service) Todo(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.Todo(ctx, actor, page, pageSize)
}

func (s *Service) Done(ctx context.Context, actor domain.Actor, page, pageSize int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.Done(ctx, actor, page, pageSize)
}

func (s *Service) Decide(ctx context.Context, actor domain.Actor, input domain.DecisionInput) (domain.Task, domain.Instance, error) {
	if input.Action != "approve" && input.Action != "reject" {
		return domain.Task{}, domain.Instance{}, apperror.Validation("action", "action must be approve or reject.")
	}
	if len(input.Comment) > 2000 {
		return domain.Task{}, domain.Instance{}, apperror.Validation("comment", "Comment must not exceed 2000 characters.")
	}
	_, instance, definition, node, err := s.repo.TaskContext(ctx, input.TaskID, actor)
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
	return s.repo.Decide(ctx, actor, input, decision)
}

func (s *Service) Terminate(ctx context.Context, instanceID, actorID uuid.UUID, comment string) (domain.Instance, error) {
	if len(comment) > 2000 {
		return domain.Instance{}, apperror.Validation("comment", "Comment must not exceed 2000 characters.")
	}
	instance, err := s.repo.Terminate(ctx, instanceID, actorID, comment)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Instance{}, apperror.Conflict("Only a running workflow instance can be terminated.")
	}
	return instance, err
}

func (s *Service) Events(ctx context.Context, instanceID uuid.UUID) ([]domain.Event, error) {
	return s.repo.Events(ctx, instanceID)
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
