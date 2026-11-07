package postgres

import (
	"context"
	"encoding/hex"

	sqlc "mars-admin-go/internal/modules/storage/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/storage/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, file domain.File) (domain.File, error) {
	hash, _ := hex.DecodeString(file.SHA256)
	row, err := sqlc.New(r.pool).CreateFile(ctx, sqlc.CreateFileParams{
		ID:              pgUUID(file.ID),
		GroupID:         optionalUUID(file.GroupID),
		OwnerUserID:     optionalUUID(file.OwnerUserID),
		OriginalName:    file.OriginalName,
		StorageKey:      file.StorageKey,
		StorageProvider: file.StorageProvider,
		ContentType:     file.ContentType,
		SizeBytes:       file.SizeBytes,
		Sha256Hash:      hash,
		Visibility:      file.Visibility,
	})
	if err != nil {
		return domain.File{}, err
	}
	return mapFile(row), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.File, error) {
	row, err := sqlc.New(r.pool).GetFile(ctx, pgUUID(id))
	if err != nil {
		return domain.File{}, err
	}
	return mapFile(row), nil
}

func (r *Repository) List(ctx context.Context, groupID *uuid.UUID, query string, page, pageSize int) ([]domain.File, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListFiles(ctx, sqlc.ListFilesParams{
		Column1: optionalUUID(groupID), Column2: query,
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountFiles(ctx, sqlc.CountFilesParams{Column1: optionalUUID(groupID), Column2: query})
	if err != nil {
		return nil, 0, err
	}
	files := make([]domain.File, 0, len(rows))
	for _, row := range rows {
		files = append(files, mapFile(row))
	}
	return files, total, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := sqlc.New(r.pool).SoftDeleteFile(ctx, pgUUID(id))
	return err
}

func (r *Repository) FileGroups(ctx context.Context) ([]domain.FileGroup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.name, g.sort_order, count(f.id), g.created_at
		FROM storage_file_groups g
		LEFT JOIN storage_files f ON f.group_id = g.id AND f.deleted_at IS NULL
		GROUP BY g.id, g.name, g.sort_order, g.created_at
		ORDER BY g.sort_order, g.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.FileGroup, 0)
	for rows.Next() {
		var item domain.FileGroup
		if err := rows.Scan(&item.ID, &item.Name, &item.Sort, &item.FileCount, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) FileGroup(ctx context.Context, id uuid.UUID) (domain.FileGroup, error) {
	var item domain.FileGroup
	err := r.pool.QueryRow(ctx, `
		SELECT g.id, g.name, g.sort_order, count(f.id), g.created_at
		FROM storage_file_groups g
		LEFT JOIN storage_files f ON f.group_id = g.id AND f.deleted_at IS NULL
		WHERE g.id = $1
		GROUP BY g.id, g.name, g.sort_order, g.created_at`, id).
		Scan(&item.ID, &item.Name, &item.Sort, &item.FileCount, &item.CreatedAt)
	return item, err
}

func (r *Repository) CreateFileGroup(ctx context.Context, group domain.FileGroup) (domain.FileGroup, error) {
	var item domain.FileGroup
	err := r.pool.QueryRow(ctx, `
		INSERT INTO storage_file_groups (id, name, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, name, sort_order, 0, created_at`, group.ID, group.Name, group.Sort).
		Scan(&item.ID, &item.Name, &item.Sort, &item.FileCount, &item.CreatedAt)
	return item, err
}

func (r *Repository) UpdateFileGroup(ctx context.Context, group domain.FileGroup) (domain.FileGroup, error) {
	var item domain.FileGroup
	err := r.pool.QueryRow(ctx, `
		UPDATE storage_file_groups
		SET name = $2, sort_order = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, name, sort_order, 0, created_at`, group.ID, group.Name, group.Sort).
		Scan(&item.ID, &item.Name, &item.Sort, &item.FileCount, &item.CreatedAt)
	return item, err
}

func (r *Repository) DeleteFileGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM storage_file_groups WHERE id = $1`, id)
	return err
}

func mapFile(row sqlc.StorageFile) domain.File {
	file := domain.File{
		ID: uuid.UUID(row.ID.Bytes), OriginalName: row.OriginalName, StorageKey: row.StorageKey,
		StorageProvider: row.StorageProvider, ContentType: row.ContentType, SizeBytes: row.SizeBytes,
		SHA256: hex.EncodeToString(row.Sha256Hash), Visibility: row.Visibility, Status: row.Status,
		CreatedAt: row.CreatedAt.Time.UTC(),
	}
	if row.GroupID.Valid {
		id := uuid.UUID(row.GroupID.Bytes)
		file.GroupID = &id
	}
	if row.OwnerUserID.Valid {
		id := uuid.UUID(row.OwnerUserID.Bytes)
		file.OwnerUserID = &id
	}
	return file
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
