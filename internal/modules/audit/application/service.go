package application

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mars-admin-go/internal/modules/audit/domain"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/google/uuid"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func New(repo domain.Repository, loggers ...*slog.Logger) *Service {
	service := &Service{repo: repo}
	if len(loggers) > 0 {
		service.logger = loggers[0]
	}
	return service
}

func (s *Service) Write(ctx context.Context, action, resourceType, resourceID, outcome string, metadata map[string]any) {
	actor, _ := requestctx.ActorFrom(ctx)
	event := domain.Event{
		ID:           mustID(),
		OccurredAt:   time.Now().UTC(),
		ActorName:    actor.DisplayName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
		RequestID:    requestctx.RequestID(ctx),
		TraceID:      requestctx.TraceID(ctx),
		Metadata:     sanitize(metadata),
	}
	if actor.UserID != uuid.Nil {
		event.ActorUserID = &actor.UserID
	}
	if err := s.repo.Insert(ctx, event); err != nil && s.logger != nil {
		s.logger.ErrorContext(ctx, "persist audit event", "error", err, "action", action)
	}
}

func (s *Service) WriteHTTP(ctx context.Context, r *http.Request, action, resourceType, resourceID string, metadata map[string]any) {
	s.WriteHTTPOutcome(ctx, r, action, resourceType, resourceID, "success", metadata)
}

func (s *Service) WriteHTTPOutcome(ctx context.Context, r *http.Request, action, resourceType, resourceID, outcome string, metadata map[string]any) {
	actor, _ := requestctx.ActorFrom(ctx)
	event := domain.Event{
		ID:           mustID(),
		OccurredAt:   time.Now().UTC(),
		ActorName:    actor.DisplayName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
		RequestID:    requestctx.RequestID(ctx),
		TraceID:      requestctx.TraceID(ctx),
		IPAddress:    clientAddr(r),
		UserAgent:    r.UserAgent(),
		Metadata:     sanitize(metadata),
	}
	if actor.UserID != uuid.Nil {
		event.ActorUserID = &actor.UserID
	}
	if err := s.repo.Insert(ctx, event); err != nil && s.logger != nil {
		s.logger.ErrorContext(ctx, "persist HTTP audit event", "error", err, "action", action)
	}
}

func (s *Service) List(ctx context.Context, filter domain.Filter) ([]domain.Event, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) LoginEvents(ctx context.Context, page, pageSize int) ([]domain.LoginEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListLogin(ctx, page, pageSize)
}

func sanitize(metadata map[string]any) map[string]any {
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") || strings.Contains(lower, "private_key") {
			out[key] = "[redacted]"
			continue
		}
		switch typed := value.(type) {
		case string:
			if len(typed) > 1000 {
				out[key] = typed[:1000]
			} else {
				out[key] = typed
			}
		case map[string]any:
			out[key] = sanitize(typed)
		case []any:
			out[key] = sanitizeSlice(typed)
		case []map[string]any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				items = append(items, sanitize(item))
			}
			out[key] = items
		default:
			if encoded, err := json.Marshal(typed); err == nil && len(encoded) <= 4000 {
				out[key] = typed
			}
		}
	}
	return out
}

func sanitizeSlice(values []any) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case map[string]any:
			out = append(out, sanitize(typed))
		case []any:
			out = append(out, sanitizeSlice(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func clientAddr(r *http.Request) string {
	host := r.RemoteAddr
	if index := strings.LastIndex(host, ":"); index > 0 {
		host = host[:index]
	}
	return host
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
