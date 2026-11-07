package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID           uuid.UUID      `json:"id"`
	OccurredAt   time.Time      `json:"occurredAt"`
	ActorUserID  *uuid.UUID     `json:"actorUserId,omitempty"`
	ActorName    string         `json:"actorName"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Outcome      string         `json:"outcome"`
	RequestID    string         `json:"requestId,omitempty"`
	TraceID      string         `json:"traceId,omitempty"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Metadata     map[string]any `json:"metadata"`
}

type LoginEvent struct {
	ID            uuid.UUID  `json:"id"`
	UserID        *uuid.UUID `json:"userId,omitempty"`
	Username      string     `json:"username"`
	ClientType    string     `json:"clientType"`
	Outcome       string     `json:"outcome"`
	FailureReason string     `json:"failureReason,omitempty"`
	IPAddress     string     `json:"ipAddress,omitempty"`
	UserAgent     string     `json:"userAgent,omitempty"`
	OccurredAt    time.Time  `json:"occurredAt"`
}

type Filter struct {
	Action       string
	ResourceType string
	From         *time.Time
	To           *time.Time
	Page         int
	PageSize     int
}

type Repository interface {
	Insert(ctx context.Context, event Event) error
	List(ctx context.Context, filter Filter) ([]Event, int64, error)
	ListLogin(ctx context.Context, page, pageSize int) ([]LoginEvent, int64, error)
}
