package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sqlc "mars-admin-go/internal/modules/organization/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/organization/domain"

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

func (r *Repository) Departments(ctx context.Context, actorID uuid.UUID, dataScope string) ([]domain.Department, error) {
	rows, err := sqlc.New(r.pool).ListDepartments(ctx, sqlc.ListDepartmentsParams{
		UserID: pgUUID(actorID), Column2: dataScope,
	})
	if err != nil {
		return nil, err
	}
	items := make([]domain.Department, 0, len(rows))
	for _, row := range rows {
		department := mapDepartment(row.ID, row.ParentID, row.Name, row.Code, row.Path, row.LeaderName, row.Phone, row.Email, row.SortOrder, row.Status, row.CreatedAt)
		department.Depth = row.Depth
		items = append(items, department)
	}
	return items, nil
}

func (r *Repository) Department(ctx context.Context, id, actorID uuid.UUID, dataScope string) (domain.Department, error) {
	row, err := sqlc.New(r.pool).GetDepartment(ctx, sqlc.GetDepartmentParams{
		ID: pgUUID(id), UserID: pgUUID(actorID), Column3: dataScope,
	})
	if err != nil {
		return domain.Department{}, err
	}
	return mapDepartment(row.ID, row.ParentID, row.Name, row.Code, row.Path, row.LeaderName, row.Phone, row.Email, row.SortOrder, row.Status, row.CreatedAt), nil
}

func (r *Repository) CreateDepartment(ctx context.Context, input domain.DepartmentInput) (domain.Department, error) {
	path, err := r.makePath(ctx, r.pool, input.ParentID, input.Code)
	if err != nil {
		return domain.Department{}, err
	}
	row, err := sqlc.New(r.pool).CreateDepartment(ctx, sqlc.CreateDepartmentParams{
		ID:         pgUUID(mustID()),
		ParentID:   pgOptionalUUID(input.ParentID),
		Name:       input.Name,
		Code:       input.Code,
		Path:       path,
		LeaderName: input.LeaderName,
		Phone:      input.Phone,
		Email:      pgText(input.Email),
		SortOrder:  input.Sort,
		Status:     input.Status,
	})
	if err != nil {
		return domain.Department{}, mapConflict(err)
	}
	return mapDepartment(row.ID, row.ParentID, row.Name, row.Code, row.Path, row.LeaderName, row.Phone, row.Email, row.SortOrder, row.Status, row.CreatedAt), nil
}

func (r *Repository) UpdateDepartment(ctx context.Context, id uuid.UUID, input domain.DepartmentInput) (domain.Department, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Department{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldPath string
	if err := tx.QueryRow(ctx, `SELECT path::text FROM organization_departments WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&oldPath); err != nil {
		return domain.Department{}, err
	}
	newPath, err := r.makePath(ctx, tx, input.ParentID, input.Code)
	if err != nil {
		return domain.Department{}, err
	}
	if input.ParentID != nil {
		var parentPath string
		if err := tx.QueryRow(ctx, `SELECT path::text FROM organization_departments WHERE id = $1 AND deleted_at IS NULL`, *input.ParentID).Scan(&parentPath); err != nil {
			return domain.Department{}, err
		}
		if strings.HasPrefix(parentPath+".", oldPath+".") {
			return domain.Department{}, errors.New("department hierarchy cannot contain a cycle")
		}
	}
	row, err := sqlc.New(tx).UpdateDepartment(ctx, sqlc.UpdateDepartmentParams{
		ID:         pgUUID(id),
		ParentID:   pgOptionalUUID(input.ParentID),
		Name:       input.Name,
		Code:       input.Code,
		Path:       newPath,
		LeaderName: input.LeaderName,
		Phone:      input.Phone,
		Email:      pgText(input.Email),
		SortOrder:  input.Sort,
		Status:     input.Status,
	})
	if err != nil {
		return domain.Department{}, mapConflict(err)
	}
	if newPath != oldPath {
		if _, err := tx.Exec(ctx, `
			UPDATE organization_departments
			SET path = $3::ltree || subpath(path, nlevel($2::ltree)), updated_at = now()
			WHERE path <@ $2::ltree AND id <> $1`,
			id, oldPath, newPath,
		); err != nil {
			return domain.Department{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Department{}, err
	}
	return mapDepartment(row.ID, row.ParentID, row.Name, row.Code, row.Path, row.LeaderName, row.Phone, row.Email, row.SortOrder, row.Status, row.CreatedAt), nil
}

func (r *Repository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).SoftDeleteDepartment(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Positions(ctx context.Context) ([]domain.Position, error) {
	rows, err := sqlc.New(r.pool).ListPositions(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Position, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapPosition(row))
	}
	return items, nil
}

func (r *Repository) CreatePosition(ctx context.Context, input domain.PositionInput) (domain.Position, error) {
	row, err := sqlc.New(r.pool).CreatePosition(ctx, sqlc.CreatePositionParams{
		ID:          pgUUID(mustID()),
		ParentID:    pgOptionalUUID(input.ParentID),
		Code:        input.Code,
		Name:        input.Name,
		SortOrder:   input.Sort,
		Status:      input.Status,
		Description: input.Description,
	})
	if err != nil {
		return domain.Position{}, mapConflict(err)
	}
	return mapPosition(row), nil
}

func (r *Repository) UpdatePosition(ctx context.Context, id uuid.UUID, input domain.PositionInput) (domain.Position, error) {
	row, err := sqlc.New(r.pool).UpdatePosition(ctx, sqlc.UpdatePositionParams{
		ID:          pgUUID(id),
		ParentID:    pgOptionalUUID(input.ParentID),
		Code:        input.Code,
		Name:        input.Name,
		SortOrder:   input.Sort,
		Status:      input.Status,
		Description: input.Description,
	})
	if err != nil {
		return domain.Position{}, err
	}
	return mapPosition(row), nil
}

func (r *Repository) DeletePosition(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).SoftDeletePosition(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) makePath(ctx context.Context, query queryer, parentID *uuid.UUID, code string) (string, error) {
	label := pathLabel(code)
	if label == "" {
		return "", errors.New("department code contains no valid path characters")
	}
	if parentID == nil {
		return label, nil
	}
	var parentPath string
	if err := query.QueryRow(ctx, `SELECT path::text FROM organization_departments WHERE id = $1 AND deleted_at IS NULL`, *parentID).Scan(&parentPath); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("parent department not found")
		}
		return "", err
	}
	return parentPath + "." + label, nil
}

func pathLabel(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(value) {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && builder.Len() > 0 {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func mapDepartment(id, parentID pgtype.UUID, name, code, path, leader, phone string, email pgtype.Text, sort int32, status string, createdAt pgtype.Timestamptz) domain.Department {
	department := domain.Department{
		ID:         uuid.UUID(id.Bytes),
		Name:       name,
		Code:       code,
		Path:       path,
		LeaderName: leader,
		Phone:      phone,
		Email:      text(email),
		Sort:       sort,
		Status:     status,
		CreatedAt:  createdAt.Time.UTC(),
	}
	if parentID.Valid {
		value := uuid.UUID(parentID.Bytes)
		department.ParentID = &value
	}
	return department
}

func mapPosition(row sqlc.OrganizationPosition) domain.Position {
	position := domain.Position{
		ID:          uuid.UUID(row.ID.Bytes),
		Code:        row.Code,
		Name:        row.Name,
		Sort:        row.SortOrder,
		Status:      row.Status,
		Description: row.Description,
		CreatedAt:   row.CreatedAt.Time.UTC(),
	}
	if row.ParentID.Valid {
		value := uuid.UUID(row.ParentID.Bytes)
		position.ParentID = &value
	}
	return position
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgOptionalUUID(id *uuid.UUID) pgtype.UUID {
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

func mapConflict(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return fmt.Errorf("department or position code already exists: %w", err)
	}
	return err
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
