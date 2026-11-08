package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Notice struct {
	ID          uuid.UUID   `json:"id"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	NoticeType  int16       `json:"noticeType"`
	Channels    []string    `json:"channels"`
	TargetType  string      `json:"targetType"`
	TargetIDs   []uuid.UUID `json:"targetIds"`
	Status      string      `json:"status"`
	CreatedBy   *uuid.UUID  `json:"createdBy,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
	ReadAt      *time.Time  `json:"readAt,omitempty"`
}

type Input struct {
	Title      string      `json:"title"`
	Content    string      `json:"content"`
	NoticeType int16       `json:"noticeType"`
	Channels   []string    `json:"channels"`
	TargetType string      `json:"targetType"`
	TargetIDs  []uuid.UUID `json:"targetIds"`
}

type Repository interface {
	Create(ctx context.Context, notice Notice) (Notice, error)
	Update(ctx context.Context, id uuid.UUID, notice Notice) (Notice, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (Notice, error)
	List(ctx context.Context, status string, page, pageSize int) ([]Notice, int64, error)
	Publish(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	Withdraw(ctx context.Context, id uuid.UUID) error
	My(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]Notice, error)
	MyCount(ctx context.Context, userID uuid.UUID) (int64, error)
	GetMy(ctx context.Context, noticeID, userID uuid.UUID) (Notice, error)
	MarkRead(ctx context.Context, noticeID, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	Unread(ctx context.Context, userID uuid.UUID) (int64, error)
}

type DeliveryEnqueuer interface {
	Enqueue(ctx context.Context, noticeID uuid.UUID, channel string) (int64, error)
}
