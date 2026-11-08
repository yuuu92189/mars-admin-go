package application

import (
	"context"
	"errors"
	"strings"

	"mars-admin-go/internal/modules/access/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RoleRepository interface {
	Roles(ctx context.Context, query string, page, pageSize int) ([]domain.Role, int64, error)
	Role(ctx context.Context, id uuid.UUID) (domain.Role, error)
	CreateRole(ctx context.Context, input domain.CreateRoleInput) (domain.Role, error)
	UpdateRole(ctx context.Context, id uuid.UUID, input domain.UpdateRoleInput) (domain.Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error
	Permissions(ctx context.Context) ([]domain.Permission, error)
	SetRolePermissions(ctx context.Context, roleID uuid.UUID, permissionCodes []string) error
	SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID) error
	RoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	RolePermissionCodes(ctx context.Context, roleID uuid.UUID) ([]string, error)
}

type MenuRepository interface {
	Menus(ctx context.Context, userID uuid.UUID, superAdmin bool) ([]domain.Menu, error)
	MenuIsDescendant(ctx context.Context, menuID, descendantID uuid.UUID) (bool, error)
	CreateMenu(ctx context.Context, input domain.MenuInput) (domain.Menu, error)
	UpdateMenu(ctx context.Context, id uuid.UUID, input domain.MenuInput) (domain.Menu, error)
	DeleteMenu(ctx context.Context, id uuid.UUID) error
}

type Repository interface {
	RoleRepository
	MenuRepository
}

type Service struct {
	roles RoleRepository
	menus MenuRepository
	cache domain.Cache
}

func New(repo Repository, caches ...domain.Cache) *Service {
	service := &Service{roles: repo, menus: repo}
	if len(caches) > 0 {
		service.cache = caches[0]
	}
	return service
}

func (s *Service) Menus(ctx context.Context, userID uuid.UUID, superAdmin bool) ([]domain.Menu, error) {
	return s.menus.Menus(ctx, userID, superAdmin)
}

func (s *Service) Roles(ctx context.Context, query string, page, pageSize int) ([]domain.Role, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.roles.Roles(ctx, strings.TrimSpace(query), page, pageSize)
}

func (s *Service) Role(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	role, err := s.roles.Role(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Role{}, apperror.NotFound("Role")
	}
	return role, err
}

func (s *Service) CreateRole(ctx context.Context, actor requestctx.Actor, input domain.CreateRoleInput) (domain.Role, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	if input.Name == "" {
		return domain.Role{}, apperror.Validation("name", "Role name is required.")
	}
	if input.Code == "" || len(input.Code) > 80 {
		return domain.Role{}, apperror.Validation("code", "Role code is required and must not exceed 80 characters.")
	}
	if !validScope(input.DataScope) {
		return domain.Role{}, apperror.Validation("dataScope", "Unsupported data scope.")
	}
	if !scopeAllowed(actor, input.DataScope) {
		return domain.Role{}, apperror.Forbidden("A role cannot have a broader data scope than the administrator who creates it.")
	}
	role, err := s.roles.CreateRole(ctx, input)
	s.invalidateAccess(ctx)
	return role, err
}

func (s *Service) UpdateRole(ctx context.Context, actor requestctx.Actor, id uuid.UUID, input domain.UpdateRoleInput) (domain.Role, error) {
	if err := s.authorizeRoleMutation(ctx, actor, id); err != nil {
		return domain.Role{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return domain.Role{}, apperror.Validation("name", "Role name is required.")
	}
	if !validScope(input.DataScope) {
		return domain.Role{}, apperror.Validation("dataScope", "Unsupported data scope.")
	}
	if !scopeAllowed(actor, input.DataScope) {
		return domain.Role{}, apperror.Forbidden("A role cannot have a broader data scope than the administrator who updates it.")
	}
	if input.Status != "active" && input.Status != "disabled" {
		return domain.Role{}, apperror.Validation("status", "status must be active or disabled.")
	}
	role, err := s.roles.UpdateRole(ctx, id, input)
	s.invalidateAccess(ctx)
	return role, err
}

func (s *Service) DeleteRole(ctx context.Context, actor requestctx.Actor, id uuid.UUID) error {
	if err := s.authorizeRoleMutation(ctx, actor, id); err != nil {
		return err
	}
	err := s.roles.DeleteRole(ctx, id)
	s.invalidateAccess(ctx)
	return err
}

func (s *Service) Permissions(ctx context.Context) ([]domain.Permission, error) {
	return s.roles.Permissions(ctx)
}

func (s *Service) SetRolePermissions(ctx context.Context, actor requestctx.Actor, roleID uuid.UUID, codes []string) error {
	if err := s.authorizeRoleMutation(ctx, actor, roleID); err != nil {
		return err
	}
	if !actor.IsSuperAdmin {
		allowed := make(map[string]struct{}, len(actor.Permissions))
		for _, permission := range actor.Permissions {
			allowed[permission] = struct{}{}
		}
		for _, code := range codes {
			if _, ok := allowed[strings.TrimSpace(code)]; !ok {
				return apperror.Forbidden("A non-super administrator cannot grant permissions they do not hold.")
			}
		}
	}
	for i, code := range codes {
		codes[i] = strings.TrimSpace(code)
		if codes[i] == "" {
			return apperror.Validation("permissions", "Permission codes must not be empty.")
		}
	}
	err := s.roles.SetRolePermissions(ctx, roleID, codes)
	s.invalidateAccess(ctx)
	return err
}

func (s *Service) SetRoleMenus(ctx context.Context, actor requestctx.Actor, roleID uuid.UUID, menuIDs []uuid.UUID) error {
	if err := s.authorizeRoleMutation(ctx, actor, roleID); err != nil {
		return err
	}
	err := s.roles.SetRoleMenus(ctx, roleID, menuIDs)
	s.invalidateAccess(ctx)
	return err
}

func (s *Service) RoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return s.roles.RoleMenuIDs(ctx, roleID)
}

func (s *Service) CreateMenu(ctx context.Context, input domain.MenuInput) (domain.Menu, error) {
	if err := validateMenu(&input); err != nil {
		return domain.Menu{}, err
	}
	menu, err := s.menus.CreateMenu(ctx, input)
	s.invalidateAccess(ctx)
	return menu, err
}

func (s *Service) UpdateMenu(ctx context.Context, id uuid.UUID, input domain.MenuInput) (domain.Menu, error) {
	if err := validateMenu(&input); err != nil {
		return domain.Menu{}, err
	}
	if input.ParentID != nil && *input.ParentID == id {
		return domain.Menu{}, apperror.Validation("parentId", "A menu cannot be its own parent.")
	}
	if input.ParentID != nil {
		descendant, err := s.menus.MenuIsDescendant(ctx, id, *input.ParentID)
		if err != nil {
			return domain.Menu{}, err
		}
		if descendant {
			return domain.Menu{}, apperror.Validation("parentId", "A menu hierarchy cannot contain a cycle.")
		}
	}
	menu, err := s.menus.UpdateMenu(ctx, id, input)
	s.invalidateAccess(ctx)
	return menu, err
}

func (s *Service) DeleteMenu(ctx context.Context, id uuid.UUID) error {
	err := s.menus.DeleteMenu(ctx, id)
	s.invalidateAccess(ctx)
	return err
}

func (s *Service) invalidateAccess(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.DeletePrefix(ctx, "access:")
	}
}

func (s *Service) authorizeRoleMutation(ctx context.Context, actor requestctx.Actor, roleID uuid.UUID) error {
	if actor.IsSuperAdmin {
		return nil
	}
	role, err := s.roles.Role(ctx, roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NotFound("Role")
	}
	if err != nil {
		return err
	}
	if role.IsSuperAdmin {
		return apperror.Forbidden("Only a super administrator can modify the super administrator role.")
	}
	rolePermissions, err := s.roles.RolePermissionCodes(ctx, roleID)
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(actor.Permissions))
	for _, permission := range actor.Permissions {
		allowed[permission] = struct{}{}
	}
	for _, permission := range rolePermissions {
		if _, ok := allowed[permission]; !ok {
			return apperror.Forbidden("A non-super administrator cannot modify a role with permissions they do not hold.")
		}
	}
	return nil
}

func validateMenu(input *domain.MenuInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 80 {
		return apperror.Validation("name", "Menu name is required and must not exceed 80 characters.")
	}
	if input.Type < 1 || input.Type > 3 {
		return apperror.Validation("type", "Menu type must be 1, 2, or 3.")
	}
	return nil
}

func validScope(value string) bool {
	switch value {
	case "all", "custom", "department", "department_tree", "self":
		return true
	default:
		return false
	}
}

func scopeAllowed(actor requestctx.Actor, target string) bool {
	if actor.IsSuperAdmin {
		return true
	}
	return scopeRank(target) <= scopeRank(actor.DataScope)
}

func scopeRank(value string) int {
	switch value {
	case "department":
		return 2
	case "department_tree":
		return 3
	case "custom":
		return 4
	case "all":
		return 5
	default:
		return 1
	}
}
