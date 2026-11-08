package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID         uuid.UUID    `json:"id"`
	ParentID   *uuid.UUID   `json:"parentId,omitempty"`
	Name       string       `json:"name"`
	Code       string       `json:"code"`
	Path       string       `json:"path"`
	LeaderName string       `json:"leaderName"`
	Phone      string       `json:"phone"`
	Email      string       `json:"email,omitempty"`
	Sort       int32        `json:"sort"`
	Status     string       `json:"status"`
	Depth      int32        `json:"depth"`
	CreatedAt  time.Time    `json:"createdAt"`
	Children   []Department `json:"children,omitempty"`
}

type Position struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Sort        int32      `json:"sort"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type DepartmentInput struct {
	ParentID   *uuid.UUID `json:"parentId"`
	Name       string     `json:"name"`
	Code       string     `json:"code"`
	LeaderName string     `json:"leaderName"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Sort       int32      `json:"sort"`
	Status     string     `json:"status"`
}

type PositionInput struct {
	ParentID    *uuid.UUID `json:"parentId"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Sort        int32      `json:"sort"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
}

type Repository interface {
	Departments(ctx context.Context, actorID uuid.UUID, dataScope string) ([]Department, error)
	Department(ctx context.Context, id, actorID uuid.UUID, dataScope string) (Department, error)
	CreateDepartment(ctx context.Context, input DepartmentInput) (Department, error)
	UpdateDepartment(ctx context.Context, id uuid.UUID, input DepartmentInput) (Department, error)
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
	Positions(ctx context.Context) ([]Position, error)
	CreatePosition(ctx context.Context, input PositionInput) (Position, error)
	UpdatePosition(ctx context.Context, id uuid.UUID, input PositionInput) (Position, error)
	DeletePosition(ctx context.Context, id uuid.UUID) error
}
