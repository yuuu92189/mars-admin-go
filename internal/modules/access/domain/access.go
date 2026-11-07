package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Menu struct {
	ID         uuid.UUID  `json:"id"`
	ParentID   *uuid.UUID `json:"parentId,omitempty"`
	Name       string     `json:"name"`
	Type       int16      `json:"type"`
	Path       string     `json:"path,omitempty"`
	Component  string     `json:"component,omitempty"`
	Permission string     `json:"permission,omitempty"`
	Icon       string     `json:"icon,omitempty"`
	Sort       int32      `json:"sort"`
	Visible    bool       `json:"visible"`
	IsFrame    bool       `json:"isFrame"`
	Children   []Menu     `json:"children,omitempty"`
}

type Role struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Description  string    `json:"description"`
	IsSuperAdmin bool      `json:"isSuperAdmin"`
	DataScope    string    `json:"dataScope"`
	Status       string    `json:"status"`
	UserCount    int64     `json:"userCount"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
}

type CreateRoleInput struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	DataScope   string `json:"dataScope"`
}

type UpdateRoleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DataScope   string `json:"dataScope"`
	Status      string `json:"status"`
}

type MenuInput struct {
	ParentID   *uuid.UUID `json:"parentId"`
	Name       string     `json:"name"`
	Type       int16      `json:"type"`
	Path       string     `json:"path"`
	Component  string     `json:"component"`
	Permission string     `json:"permission"`
	Icon       string     `json:"icon"`
	Sort       int32      `json:"sort"`
	Visible    bool       `json:"visible"`
	IsFrame    bool       `json:"isFrame"`
}

type Repository interface {
	Menus(ctx context.Context, userID uuid.UUID, superAdmin bool) ([]Menu, error)
	Roles(ctx context.Context, query string, page, pageSize int) ([]Role, int64, error)
	Role(ctx context.Context, id uuid.UUID) (Role, error)
	CreateRole(ctx context.Context, input CreateRoleInput) (Role, error)
	UpdateRole(ctx context.Context, id uuid.UUID, input UpdateRoleInput) (Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error
	Permissions(ctx context.Context) ([]Permission, error)
	SetRolePermissions(ctx context.Context, roleID uuid.UUID, permissionCodes []string) error
	SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID) error
	RoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	CreateMenu(ctx context.Context, input MenuInput) (Menu, error)
	UpdateMenu(ctx context.Context, id uuid.UUID, input MenuInput) (Menu, error)
	DeleteMenu(ctx context.Context, id uuid.UUID) error
}

type Cache interface {
	DeletePrefix(ctx context.Context, suffix string) error
}
