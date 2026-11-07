package application

import (
	"context"
	"errors"
	"strings"

	"mars-admin-go/internal/modules/organization/domain"
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

func (s *Service) Departments(ctx context.Context) ([]domain.Department, error) {
	items, err := s.repo.Departments(ctx)
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
	for i := range nodes {
		nodes[i].Children = buildDepartmentTree(children[nodes[i].ID], children)
	}
	return nodes
}

func (s *Service) Department(ctx context.Context, id uuid.UUID) (domain.Department, error) {
	department, err := s.repo.Department(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Department{}, apperror.NotFound("Department")
	}
	return department, err
}

func (s *Service) CreateDepartment(ctx context.Context, input domain.DepartmentInput) (domain.Department, error) {
	if err := validateDepartmentInput(&input); err != nil {
		return domain.Department{}, err
	}
	return s.repo.CreateDepartment(ctx, input)
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, input domain.DepartmentInput) (domain.Department, error) {
	if err := validateDepartmentInput(&input); err != nil {
		return domain.Department{}, err
	}
	if input.ParentID != nil && *input.ParentID == id {
		return domain.Department{}, apperror.Validation("parentId", "A department cannot be its own parent.")
	}
	department, err := s.repo.UpdateDepartment(ctx, id, input)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Department{}, apperror.NotFound("Department")
	}
	return department, err
}

func (s *Service) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteDepartment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Conflict("The department has children or no longer exists.")
	}
	return err
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
