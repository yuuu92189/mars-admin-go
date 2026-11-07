package postgres

import (
	"context"
	"encoding/json"
	"errors"

	sqlc "mars-admin-go/internal/modules/messaging/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/notification/domain"
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

func (r *Repository) Create(ctx context.Context, notice domain.Notice) (domain.Notice, error) {
	channels, _ := json.Marshal(notice.Channels)
	targets, _ := json.Marshal(uuidStrings(notice.TargetIDs))
	row, err := sqlc.New(r.pool).CreateNotice(ctx, sqlc.CreateNoticeParams{
		ID: pgUUID(notice.ID), Title: notice.Title, Content: notice.Content,
		NoticeType: notice.NoticeType, Channels: channels, TargetType: notice.TargetType,
		TargetIds: targets, Status: "draft", CreatedBy: optionalUUID(notice.CreatedBy),
	})
	if err != nil {
		return domain.Notice{}, err
	}
	return mapNotice(row), nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, notice domain.Notice) (domain.Notice, error) {
	channels, _ := json.Marshal(notice.Channels)
	targets, _ := json.Marshal(uuidStrings(notice.TargetIDs))
	row, err := sqlc.New(r.pool).UpdateNotice(ctx, sqlc.UpdateNoticeParams{
		ID: pgUUID(id), Title: notice.Title, Content: notice.Content, NoticeType: notice.NoticeType,
		Channels: channels, TargetType: notice.TargetType, TargetIds: targets,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Notice{}, apperror.Conflict("Only a draft notice can be edited.")
	}
	if err != nil {
		return domain.Notice{}, err
	}
	return mapNotice(row), nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM notification_notices WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return apperror.NotFound("Notice")
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Notice, error) {
	row, err := sqlc.New(r.pool).GetNotice(ctx, pgUUID(id))
	if err != nil {
		return domain.Notice{}, err
	}
	return mapNotice(row), nil
}

func (r *Repository) List(ctx context.Context, status string, page, pageSize int) ([]domain.Notice, int64, error) {
	q := sqlc.New(r.pool)
	rows, err := q.ListNotices(ctx, sqlc.ListNoticesParams{
		Column1: status, Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountNotices(ctx, status)
	if err != nil {
		return nil, 0, err
	}
	notices := make([]domain.Notice, 0, len(rows))
	for _, row := range rows {
		notices = append(notices, mapNotice(row))
	}
	return notices, total, nil
}

func (r *Repository) Publish(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	notice, err := q.GetNotice(ctx, pgUUID(id))
	if err != nil {
		return nil, err
	}
	affected, err := q.PublishNotice(ctx, pgUUID(id))
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, pgx.ErrNoRows
	}
	recipients, err := resolveRecipients(ctx, tx, notice)
	if err != nil {
		return nil, err
	}
	for _, userID := range recipients {
		if _, err := tx.Exec(ctx, `
			INSERT INTO notification_recipients (notice_id, user_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, userID); err != nil {
			return nil, err
		}
	}
	if err := q.InsertNoticeSendLog(ctx, sqlc.InsertNoticeSendLogParams{
		ID: pgUUID(mustID()), NoticeID: pgUUID(id), Channel: "station", Status: "success",
		TargetCount: int32(len(recipients)), SuccessCount: int32(len(recipients)), ErrorMessage: "", // #nosec G115 -- recipient count is bounded by configured application scale.
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return recipients, nil
}

func (r *Repository) Withdraw(ctx context.Context, id uuid.UUID) error {
	affected, err := sqlc.New(r.pool).WithdrawNotice(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) My(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]domain.Notice, error) {
	rows, err := sqlc.New(r.pool).ListMyNotices(ctx, sqlc.ListMyNoticesParams{
		UserID: pgUUID(userID), Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), // #nosec G115 -- application layer caps page size at 100.
	})
	if err != nil {
		return nil, err
	}
	notices := make([]domain.Notice, 0, len(rows))
	for _, row := range rows {
		notice := domain.Notice{
			ID: uuid.UUID(row.ID.Bytes), Title: row.Title, Content: row.Content, NoticeType: row.NoticeType,
			TargetType: row.TargetType, Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
			UpdatedAt: row.UpdatedAt.Time.UTC(),
		}
		_ = json.Unmarshal(row.Channels, &notice.Channels)
		_ = json.Unmarshal(row.TargetIds, &notice.TargetIDs)
		if row.PublishedAt.Valid {
			value := row.PublishedAt.Time.UTC()
			notice.PublishedAt = &value
		}
		if row.ReadAt.Valid {
			value := row.ReadAt.Time.UTC()
			notice.ReadAt = &value
		}
		notices = append(notices, notice)
	}
	return notices, nil
}

func (r *Repository) GetMy(ctx context.Context, noticeID, userID uuid.UUID) (domain.Notice, error) {
	row, err := sqlc.New(r.pool).GetMyNotice(ctx, sqlc.GetMyNoticeParams{
		ID: pgUUID(noticeID), UserID: pgUUID(userID),
	})
	if err != nil {
		return domain.Notice{}, err
	}
	notice := domain.Notice{
		ID: uuid.UUID(row.ID.Bytes), Title: row.Title, Content: row.Content, NoticeType: row.NoticeType,
		TargetType: row.TargetType, Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
		UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Channels, &notice.Channels)
	_ = json.Unmarshal(row.TargetIds, &notice.TargetIDs)
	if row.PublishedAt.Valid {
		value := row.PublishedAt.Time.UTC()
		notice.PublishedAt = &value
	}
	if row.ReadAt.Valid {
		value := row.ReadAt.Time.UTC()
		notice.ReadAt = &value
	}
	return notice, nil
}

func (r *Repository) MarkRead(ctx context.Context, noticeID, userID uuid.UUID) error {
	affected, err := sqlc.New(r.pool).MarkNoticeRead(ctx, sqlc.MarkNoticeReadParams{
		NoticeID: pgUUID(noticeID), UserID: pgUUID(userID),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.NotFound("Notice")
	}
	return nil
}

func (r *Repository) Unread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return sqlc.New(r.pool).CountUnreadNotices(ctx, pgUUID(userID))
}

func resolveRecipients(ctx context.Context, tx pgx.Tx, notice sqlc.NotificationNotice) ([]uuid.UUID, error) {
	switch notice.TargetType {
	case "users":
		var strings []string
		if err := json.Unmarshal(notice.TargetIds, &strings); err != nil {
			return nil, err
		}
		return parseUUIDs(strings)
	case "departments":
		var strings []string
		if err := json.Unmarshal(notice.TargetIds, &strings); err != nil {
			return nil, err
		}
		departmentIDs, err := parseUUIDs(strings)
		if err != nil {
			return nil, err
		}
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT m.user_id
			FROM organization_memberships m
			WHERE m.department_id = ANY($1::uuid[])`, departmentIDs)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanUUIDs(rows)
	default:
		rows, err := tx.Query(ctx, `SELECT id FROM identity_users WHERE deleted_at IS NULL AND status = 'active'`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanUUIDs(rows)
	}
}

func mapNotice(row sqlc.NotificationNotice) domain.Notice {
	notice := domain.Notice{
		ID: uuid.UUID(row.ID.Bytes), Title: row.Title, Content: row.Content, NoticeType: row.NoticeType,
		TargetType: row.TargetType, Status: row.Status, CreatedAt: row.CreatedAt.Time.UTC(),
		UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Channels, &notice.Channels)
	_ = json.Unmarshal(row.TargetIds, &notice.TargetIDs)
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		notice.CreatedBy = &id
	}
	if row.PublishedAt.Valid {
		value := row.PublishedAt.Time.UTC()
		notice.PublishedAt = &value
	}
	return notice
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func scanUUIDs(rows pgx.Rows) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func uuidStrings(ids []uuid.UUID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
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

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
