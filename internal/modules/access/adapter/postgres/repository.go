package postgres

import (
	"context"
	"errors"
	"strings"

	sqlc "mars-admin-go/internal/modules/access/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/access/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Menus(ctx context.Context, userID uuid.UUID, superAdmin bool) ([]domain.Menu, error) {
	q := sqlc.New(r.pool)
	var rows []sqlc.AccessMenu
	var err error
	if superAdmin {
		rows, err = q.ListAllMenus(ctx)
	} else {
		rows, err = q.ListMenusForUser(ctx, pgUUID(userID))
	}
	if err != nil {
		return nil, err
	}
	flat := make([]domain.Menu, 0, len(rows))
	for _, row := range rows {
		menu := domain.Menu{
			ID:         uuid.UUID(row.ID.Bytes),
			Name:       row.Name,
			Type:       row.MenuType,
			Path:       text(row.RoutePath),
			Component:  text(row.Component),
			Permission: text(row.PermissionCode),
			Icon:       row.Icon,
			Sort:       row.SortOrder,
			Visible:    row.Visible,
			IsFrame:    row.ExternalUrl.Valid,
		}
		if row.ParentID.Valid {
			parentID := uuid.UUID(row.ParentID.Bytes)
			menu.ParentID = &parentID
		}
		flat = append(flat, menu)
	}
	byParent := make(map[uuid.UUID][]domain.Menu)
	roots := make([]domain.Menu, 0)
	ids := make(map[uuid.UUID]struct{}, len(flat))
	for _, menu := range flat {
		ids[menu.ID] = struct{}{}
	}
	for _, menu := range flat {
		if menu.ParentID == nil {
			roots = append(roots, menu)
			continue
		}
		if _, exists := ids[*menu.ParentID]; !exists {
			roots = append(roots, menu)
			continue
		}
		byParent[*menu.ParentID] = append(byParent[*menu.ParentID], menu)
	}
	return buildTree(roots, byParent), nil
}

func buildTree(nodes []domain.Menu, children map[uuid.UUID][]domain.Menu) []domain.Menu {
	for i := range nodes {
		nodes[i].Children = buildTree(children[nodes[i].ID], children)
	}
	return nodes
}

func (r *Repository) Roles(ctx context.Context, query string, page, pageSize int) ([]domain.Role, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListRoles(ctx, sqlc.ListRolesParams{
		Column1: query,
		Limit:   int32(pageSize),              // #nosec G115 -- application layer caps pageSize at 100.
		Offset:  int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountRoles(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, domain.Role{
			ID:           uuid.UUID(row.ID.Bytes),
			Name:         row.Name,
			Code:         row.Code,
			Description:  row.Description,
			IsSuperAdmin: row.IsSuperAdmin,
			DataScope:    row.DataScope,
			Status:       row.Status,
			UserCount:    row.UserCount,
			Version:      row.Version,
			CreatedAt:    row.CreatedAt.Time.UTC(),
		})
	}
	return roles, total, nil
}

func (r *Repository) Role(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	row, err := sqlc.New(r.pool).GetRole(ctx, pgUUID(id))
	if err != nil {
		return domain.Role{}, err
	}
	return mapRole(row), nil
}

func (r *Repository) CreateRole(ctx context.Context, input domain.CreateRoleInput) (domain.Role, error) {
	row, err := sqlc.New(r.pool).CreateRole(ctx, sqlc.CreateRoleParams{
		ID:          pgUUID(mustID()),
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
		DataScope:   input.DataScope,
	})
	if err != nil {
		return domain.Role{}, mapConflict(err)
	}
	return mapRole(row), nil
}

func (r *Repository) UpdateRole(ctx context.Context, id uuid.UUID, input domain.UpdateRoleInput) (domain.Role, error) {
	row, err := sqlc.New(r.pool).UpdateRole(ctx, sqlc.UpdateRoleParams{
		ID:          pgUUID(id),
		Name:        input.Name,
		Description: input.Description,
		DataScope:   input.DataScope,
		Status:      input.Status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Role{}, errors.New("role not found")
	}
	if err != nil {
		return domain.Role{}, err
	}
	return mapRole(row), nil
}

func (r *Repository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).SoftDeleteRole(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("role not found or super administrator cannot be deleted")
	}
	return nil
}

func (r *Repository) Permissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := sqlc.New(r.pool).ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	permissions := make([]domain.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, domain.Permission{
			ID:          uuid.UUID(row.ID.Bytes),
			Code:        row.Code,
			Name:        row.Name,
			Module:      row.Module,
			Description: row.Description,
		})
	}
	return permissions, nil
}

func (r *Repository) SetRolePermissions(ctx context.Context, roleID uuid.UUID, permissionCodes []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	if err := q.SetRolePermissions(ctx, pgUUID(roleID)); err != nil {
		return err
	}
	for _, code := range permissionCodes {
		if err := q.AddRolePermission(ctx, sqlc.AddRolePermissionParams{RoleID: pgUUID(roleID), Code: code}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	if err := q.SetRoleMenus(ctx, pgUUID(roleID)); err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if err := q.AddRoleMenu(ctx, sqlc.AddRoleMenuParams{RoleID: pgUUID(roleID), MenuID: pgUUID(menuID)}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) RoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := sqlc.New(r.pool).ListRoleMenus(ctx, pgUUID(roleID))
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, uuid.UUID(row.Bytes))
	}
	return ids, nil
}

func (r *Repository) CreateMenu(ctx context.Context, input domain.MenuInput) (domain.Menu, error) {
	row, err := sqlc.New(r.pool).CreateMenu(ctx, sqlc.CreateMenuParams{
		ID: pgUUID(mustID()), ParentID: optionalUUID(input.ParentID), Name: input.Name,
		MenuType: input.Type, RoutePath: pgText(input.Path), Component: pgText(input.Component),
		PermissionCode: pgText(input.Permission), Icon: input.Icon, SortOrder: input.Sort,
		Visible: input.Visible, ExternalUrl: pgText(externalURL(input)), Status: "active",
	})
	if err != nil {
		return domain.Menu{}, err
	}
	return mapMenu(row), nil
}

func (r *Repository) UpdateMenu(ctx context.Context, id uuid.UUID, input domain.MenuInput) (domain.Menu, error) {
	row, err := sqlc.New(r.pool).UpdateMenu(ctx, sqlc.UpdateMenuParams{
		ID: pgUUID(id), ParentID: optionalUUID(input.ParentID), Name: input.Name,
		MenuType: input.Type, RoutePath: pgText(input.Path), Component: pgText(input.Component),
		PermissionCode: pgText(input.Permission), Icon: input.Icon, SortOrder: input.Sort,
		Visible: input.Visible, ExternalUrl: pgText(externalURL(input)), Status: "active",
	})
	if err != nil {
		return domain.Menu{}, err
	}
	return mapMenu(row), nil
}

func (r *Repository) DeleteMenu(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).SoftDeleteMenu(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("menu not found or still has children")
	}
	return nil
}

func mapMenu(row sqlc.AccessMenu) domain.Menu {
	menu := domain.Menu{
		ID: uuid.UUID(row.ID.Bytes), Name: row.Name, Type: row.MenuType,
		Path: text(row.RoutePath), Component: text(row.Component), Permission: text(row.PermissionCode),
		Icon: row.Icon, Sort: row.SortOrder, Visible: row.Visible, IsFrame: row.ExternalUrl.Valid,
	}
	if row.ParentID.Valid {
		id := uuid.UUID(row.ParentID.Bytes)
		menu.ParentID = &id
	}
	return menu
}

func externalURL(input domain.MenuInput) string {
	if input.IsFrame {
		return input.Component
	}
	return ""
}

func mapRole(row sqlc.AccessRole) domain.Role {
	return domain.Role{
		ID:           uuid.UUID(row.ID.Bytes),
		Name:         row.Name,
		Code:         row.Code,
		Description:  row.Description,
		IsSuperAdmin: row.IsSuperAdmin,
		DataScope:    row.DataScope,
		Status:       row.Status,
		Version:      row.Version,
		CreatedAt:    row.CreatedAt.Time.UTC(),
	}
}

func mapConflict(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return errors.New("role code already exists")
	}
	return err
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func optionalUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func text(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
