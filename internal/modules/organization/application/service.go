package application

import (
	"context"
	"errors"
	"strings"

	"mars-admin-go/internal/modules/organization/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo   domain.Repository
	scopes ScopeResolver
}

type ScopeResolver interface {
	ResolveDataScope(ctx context.Context, userID uuid.UUID) (string, bool, error)
}

func New(repo domain.Repository, scopes ...ScopeResolver) *Service {
	service := &Service{repo: repo}
	if len(scopes) > 0 {
		service.scopes = scopes[0]
	}
	return service
}

func (s *Service) Departments(ctx context.Context, actor requestctx.Actor) ([]domain.Department, error) {
	dataScope, err := s.resolveScope(ctx, actor)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.Departments(ctx, actor.UserID, dataScope)
	if err != nil {
		return nil, err
	}
	children := make(map[uuid.UUID][]domain.Department)
	roots := make([]domain.Department, 0)
	ids := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		ids[item.ID] = struct{}{}
	}
	for _, item := range items {
		if item.ParentID == nil {
			roots = append(roots, item)
			continue
		}
		if _, ok := ids[*item.ParentID]; !ok {
			roots = append(roots, item)
			continue
		}
		children[*item.ParentID] = append(children[*item.ParentID], item)
	}
	return buildDepartmentTree(roots, children), nil
}

func buildDepartmentTree(nodes []domain.Department, children map[uuid.UUID][]domain.Department) []domain.Department {
	return buildDepartmentTreeSafe(nodes, children, make(map[uuid.UUID]bool))
}

func buildDepartmentTreeSafe(nodes []domain.Department, children map[uuid.UUID][]domain.Department, visiting map[uuid.UUID]bool) []domain.Department {
	for i := range nodes {
		if visiting[nodes[i].ID] {
			nodes[i].Children = nil
			continue
		}
		visiting[nodes[i].ID] = true
		nodes[i].Children = buildDepartmentTreeSafe(children[nodes[i].ID], children, visiting)
		delete(visiting, nodes[i].ID)
	}
	return nodes
}

func (s *Service) Department(ctx context.Context, actor requestctx.Actor, id uuid.UUID) (domain.Department, error) {
	dataScope, err := s.resolveScope(ctx, actor)
	if err != nil {
		return domain.Department{}, err
	}
	department, err := s.repo.Department(ctx, id, actor.UserID, dataScope)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Department{}, apperror.NotFound("Department")
	}
	return department, err
}

func (s *Service) CreateDepartment(ctx context.Context, actor requestctx.Actor, input domain.DepartmentInput) (domain.Department, error) {
	if err := validateDepartmentInput(&input); err != nil {
		return domain.Department{}, err
	}
	dataScope, err := s.resolveScope(ctx, actor)
	if err != nil {
		return domain.Department{}, err
	}
	if input.ParentID == nil {
		if dataScope != "all" {
			return domain.Department{}, apperror.Forbidden("Only an all-scope administrator can create a root department.")
		}
	} else if _, err := s.repo.Department(ctx, *input.ParentID, actor.UserID, dataScope); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Department{}, apperror.Forbidden("The parent department is outside your data scope.")
		}
		return domain.Department{}, err
	}
	return s.repo.CreateDepartment(ctx, input)
}

func (s *Service) UpdateDepartment(ctx context.Context, actor requestctx.Actor, id uuid.UUID, input domain.DepartmentInput) (domain.Department, error) {
	if err := validateDepartmentInput(&input); err != nil {
		return domain.Department{}, err
	}
	dataScope, err := s.resolveScope(ctx, actor)
	if err != nil {
		return domain.Department{}, err
	}
	if _, err := s.repo.Department(ctx, id, actor.UserID, dataScope); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Department{}, apperror.NotFound("Department")
		}
		return domain.Department{}, err
	}
	if input.ParentID != nil && *input.ParentID == id {
		return domain.Department{}, apperror.Validation("parentId", "A department cannot be its own parent.")
	}
	if input.ParentID == nil {
		if dataScope != "all" {
			return domain.Department{}, apperror.Forbidden("Only an all-scope administrator can move a department to root.")
		}
	} else if _, err := s.repo.Department(ctx, *input.ParentID, actor.UserID, dataScope); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Department{}, apperror.Forbidden("The parent department is outside your data scope.")
		}
		return domain.Department{}, err
	}
	department, err := s.repo.UpdateDepartment(ctx, id, input)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Department{}, apperror.NotFound("Department")
	}
	return department, err
}

func (s *Service) DeleteDepartment(ctx context.Context, actor requestctx.Actor, id uuid.UUID) error {
	dataScope, err := s.resolveScope(ctx, actor)
	if err != nil {
		return err
	}
	if _, err := s.repo.Department(ctx, id, actor.UserID, dataScope); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NotFound("Department")
		}
		return err
	}
	err = s.repo.DeleteDepartment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Conflict("The department has children or no longer exists.")
	}
	return err
}

func (s *Service) resolveScope(ctx context.Context, actor requestctx.Actor) (string, error) {
	if actor.UserID == uuid.Nil {
		return "", apperror.Unauthorized("Authentication is required.")
	}
	if s.scopes == nil {
		return "all", nil
	}
	scope, superAdmin, err := s.scopes.ResolveDataScope(ctx, actor.UserID)
	if err != nil {
		return "", err
	}
	if superAdmin {
		return "all", nil
	}
	if scope == "" {
		scope = "self"
	}
	switch scope {
	case "all", "self", "department", "department_tree", "custom":
		return scope, nil
	default:
		return "self", nil
	}
}

func (s *Service) Positions(ctx context.Context) ([]domain.Position, error) {
	return s.repo.Positions(ctx)
}

func (s *Service) CreatePosition(ctx context.Context, input domain.PositionInput) (domain.Position, error) {
	if err := validatePositionInput(&input); err != nil {
		return domain.Position{}, err
	}
	return s.repo.CreatePosition(ctx, input)
}

func (s *Service) UpdatePosition(ctx context.Context, id uuid.UUID, input domain.PositionInput) (domain.Position, error) {
	if err := validatePositionInput(&input); err != nil {
		return domain.Position{}, err
	}
	return s.repo.UpdatePosition(ctx, id, input)
}

func (s *Service) DeletePosition(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePosition(ctx, id)
}

func validateDepartmentInput(input *domain.DepartmentInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if input.Name == "" {
		return apperror.Validation("name", "Department name is required.")
	}
	if input.Code == "" || len(input.Code) > 50 {
		return apperror.Validation("code", "Department code is required and must not exceed 50 characters.")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		return apperror.Validation("status", "status must be active or disabled.")
	}
	if input.Sort < 0 || input.Sort > 100000 {
		return apperror.Validation("sort", "sort must be between 0 and 100000.")
	}
	return nil
}

func validatePositionInput(input *domain.PositionInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if input.Name == "" {
		return apperror.Validation("name", "Position name is required.")
	}
	if input.Code == "" || len(input.Code) > 80 {
		return apperror.Validation("code", "Position code is required and must not exceed 80 characters.")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		return apperror.Validation("status", "status must be active or disabled.")
	}
	return nil
}
