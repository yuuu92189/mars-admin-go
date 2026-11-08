package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	sqlc "mars-admin-go/internal/modules/system/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/system/domain"
	"mars-admin-go/internal/platform/apperror"

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

func (r *Repository) DictionaryTypes(ctx context.Context) ([]domain.DictionaryType, error) {
	rows, err := sqlc.New(r.pool).ListDictionaryTypes(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DictionaryType, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.DictionaryType{
			ID:          uuid.UUID(row.ID.Bytes),
			Code:        row.Code,
			Name:        row.Name,
			Description: row.Description,
			Status:      row.Status,
			ItemCount:   row.ItemCount,
			CreatedAt:   row.CreatedAt.Time.UTC(),
		})
	}
	return items, nil
}

func (r *Repository) DictionaryItems(ctx context.Context, typeID uuid.UUID) ([]domain.DictionaryItem, error) {
	rows, err := sqlc.New(r.pool).ListDictionaryItems(ctx, pgUUID(typeID))
	if err != nil {
		return nil, err
	}
	items := make([]domain.DictionaryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapDictionaryItem(row))
	}
	return items, nil
}

func (r *Repository) CreateDictionaryType(ctx context.Context, input domain.DictionaryTypeInput) (domain.DictionaryType, error) {
	row, err := sqlc.New(r.pool).CreateDictionaryType(ctx, sqlc.CreateDictionaryTypeParams{
		ID:          pgUUID(mustID()),
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
	})
	if err != nil {
		return domain.DictionaryType{}, mapConflict(err)
	}
	return domain.DictionaryType{
		ID: row.ID.Bytes, Code: row.Code, Name: row.Name, Description: row.Description,
		Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
	}, nil
}

func (r *Repository) UpdateDictionaryType(ctx context.Context, id uuid.UUID, input domain.DictionaryTypeInput) (domain.DictionaryType, error) {
	row, err := sqlc.New(r.pool).UpdateDictionaryType(ctx, sqlc.UpdateDictionaryTypeParams{
		ID:          pgUUID(id),
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Status:      input.Status,
	})
	if err != nil {
		return domain.DictionaryType{}, mapConflict(err)
	}
	return domain.DictionaryType{
		ID: uuid.UUID(row.ID.Bytes), Code: row.Code, Name: row.Name, Description: row.Description,
		Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
	}, nil
}

func (r *Repository) DeleteDictionaryType(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).DeleteDictionaryType(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.NotFound("Dictionary type")
	}
	return nil
}

func (r *Repository) CreateDictionaryItem(ctx context.Context, input domain.DictionaryItemInput) (domain.DictionaryItem, error) {
	row, err := sqlc.New(r.pool).CreateDictionaryItem(ctx, sqlc.CreateDictionaryItemParams{
		ID:        pgUUID(mustID()),
		TypeID:    pgUUID(input.TypeID),
		Value:     input.Value,
		Label:     input.Label,
		SortOrder: input.Sort,
		TagStyle:  input.TagStyle,
		IsDefault: input.IsDefault,
		Status:    input.Status,
	})
	if err != nil {
		return domain.DictionaryItem{}, mapConflict(err)
	}
	return mapDictionaryItem(row), nil
}

func (r *Repository) UpdateDictionaryItem(ctx context.Context, id uuid.UUID, input domain.DictionaryItemInput) (domain.DictionaryItem, error) {
	row, err := sqlc.New(r.pool).UpdateDictionaryItem(ctx, sqlc.UpdateDictionaryItemParams{
		ID:        pgUUID(id),
		Value:     input.Value,
		Label:     input.Label,
		SortOrder: input.Sort,
		TagStyle:  input.TagStyle,
		IsDefault: input.IsDefault,
		Status:    input.Status,
	})
	if err != nil {
		return domain.DictionaryItem{}, mapConflict(err)
	}
	return mapDictionaryItem(row), nil
}

func (r *Repository) DeleteDictionaryItem(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).DeleteDictionaryItem(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.NotFound("Dictionary item")
	}
	return nil
}

func (r *Repository) SettingGroups(ctx context.Context) ([]domain.SettingGroup, error) {
	rows, err := sqlc.New(r.pool).ListSettingGroups(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.SettingGroup, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.SettingGroup{
			ID: uuid.UUID(row.ID.Bytes), Code: row.Code, Name: row.Name, Description: row.Description,
			Sort: row.SortOrder, Status: row.Status, SettingCount: row.SettingCount,
		})
	}
	return items, nil
}

func (r *Repository) SettingGroupByCode(ctx context.Context, code string, includeSensitive bool) (domain.SettingGroup, error) {
	q := sqlc.New(r.pool)
	groupRow, err := q.GetSettingGroupByCode(ctx, code)
	if err != nil {
		return domain.SettingGroup{}, err
	}
	rows, err := q.ListSettingsByGroup(ctx, groupRow.ID)
	if err != nil {
		return domain.SettingGroup{}, err
	}
	group := domain.SettingGroup{
		ID: uuid.UUID(groupRow.ID.Bytes), Code: groupRow.Code, Name: groupRow.Name,
		Description: groupRow.Description, Sort: groupRow.SortOrder, Status: groupRow.Status,
		SettingCount: int64(len(rows)),
		Settings:     make([]domain.Setting, 0, len(rows)),
	}
	for _, row := range rows {
		setting := domain.Setting{
			ID: uuid.UUID(row.ID.Bytes), Key: row.Key, ValueType: row.ValueType,
			Sensitive: row.IsSensitive, Version: row.Version, UpdatedAt: row.UpdatedAt.Time.UTC(),
		}
		switch {
		case row.IsSensitive && !includeSensitive:
			setting.Value = "********"
		case row.IsSensitive:
			setting.Value = "<encrypted>"
			setting.RawValue = []byte(row.ValueText)
		default:
			setting.Value, _ = decodeValue(row.ValueType, []byte(row.ValueText))
		}
		group.Settings = append(group.Settings, setting)
	}
	return group, nil
}

func (r *Repository) PublicSettings(ctx context.Context) (map[string]any, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.code || '.' || s.key, s.value_type, s.value_text
		FROM system_settings s
		JOIN system_setting_groups g ON g.id = s.group_id
		WHERE s.is_sensitive = false AND g.status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make(map[string]any)
	for rows.Next() {
		var key, valueType, valueText string
		if err := rows.Scan(&key, &valueType, &valueText); err != nil {
			return nil, err
		}
		value, _ := decodeValue(valueType, []byte(valueText))
		values[key] = value
	}
	return values, rows.Err()
}

func (r *Repository) UpsertSettings(ctx context.Context, groupCode string, updates []domain.SettingUpdate, changedBy uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	group, err := sqlc.New(tx).GetSettingGroupByCode(ctx, groupCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NotFound("Setting group")
	}
	if err != nil {
		return err
	}
	q := sqlc.New(tx)
	validation := []byte(`{}`)
	for _, update := range updates {
		var oldValue string
		err := tx.QueryRow(ctx, `
			SELECT value_text
			FROM system_settings
			WHERE group_id = $1 AND key = $2`, group.ID, update.Key).Scan(&oldValue)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		valueType := update.ValueType
		if update.Sensitive {
			valueType = "secret"
		}
		row, err := q.UpsertSetting(ctx, sqlc.UpsertSettingParams{
			ID: pgUUID(mustID()), GroupID: group.ID, Key: update.Key, ValueType: valueType,
			ValueText: string(update.Raw), IsSensitive: update.Sensitive, Validation: validation,
		})
		if err != nil {
			return err
		}
		if err := q.RecordSettingRevision(ctx, sqlc.RecordSettingRevisionParams{
			ID: pgUUID(mustID()), SettingID: row.ID, OldValue: pgText(oldValue),
			NewValue: pgText(string(update.Raw)), ChangedBy: pgUUID(changedBy),
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func decodeValue(valueType string, raw []byte) (any, error) {
	switch valueType {
	case "integer":
		value, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil {
			return string(raw), err
		}
		return value, nil
	case "boolean":
		return string(raw) == "true", nil
	case "json":
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return string(raw), err
		}
		return value, nil
	default:
		return string(raw), nil
	}
}

func mapDictionaryItem(row sqlc.SystemDictionaryItem) domain.DictionaryItem {
	return domain.DictionaryItem{
		ID: uuid.UUID(row.ID.Bytes), TypeID: uuid.UUID(row.TypeID.Bytes), Value: row.Value,
		Label: row.Label, Sort: row.SortOrder, TagStyle: row.TagStyle,
		IsDefault: row.IsDefault, Status: row.Status,
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func mapConflict(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return apperror.Conflict("A record with the same unique value already exists.")
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
