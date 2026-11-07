package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	messagingdomain "mars-admin-go/internal/modules/messaging/domain"
	notificationdomain "mars-admin-go/internal/modules/notification/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo      notificationdomain.Repository
	publisher messagingdomain.Publisher
	delivery  notificationdomain.DeliveryEnqueuer
}

func New(repo notificationdomain.Repository, publisher messagingdomain.Publisher, deliveries ...notificationdomain.DeliveryEnqueuer) *Service {
	service := &Service{repo: repo, publisher: publisher}
	if len(deliveries) > 0 {
		service.delivery = deliveries[0]
	}
	return service
}

func (s *Service) Create(ctx context.Context, createdBy uuid.UUID, input notificationdomain.Input) (notificationdomain.Notice, error) {
	if err := normalizeInput(&input); err != nil {
		return notificationdomain.Notice{}, err
	}
	notice := notificationdomain.Notice{
		ID: mustID(), Title: input.Title, Content: input.Content, NoticeType: input.NoticeType,
		Channels: input.Channels, TargetType: input.TargetType, TargetIDs: input.TargetIDs,
		Status: "draft", CreatedBy: &createdBy, CreatedAt: time.Now().UTC(),
	}
	return s.repo.Create(ctx, notice)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input notificationdomain.Input) (notificationdomain.Notice, error) {
	if err := normalizeInput(&input); err != nil {
		return notificationdomain.Notice{}, err
	}
	notice := notificationdomain.Notice{
		Title: input.Title, Content: input.Content, NoticeType: input.NoticeType,
		Channels: input.Channels, TargetType: input.TargetType, TargetIDs: input.TargetIDs,
	}
	return s.repo.Update(ctx, id, notice)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (notificationdomain.Notice, error) {
	notice, err := s.repo.Get(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return notificationdomain.Notice{}, apperror.NotFound("Notice")
	}
	return notice, err
}

func (s *Service) List(ctx context.Context, status string, page, pageSize int) ([]notificationdomain.Notice, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, status, page, pageSize)
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) error {
	recipients, err := s.repo.Publish(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.Conflict("The notice is no longer a draft.")
		}
		return err
	}
	notice, _ := s.repo.Get(ctx, id)
	if s.delivery != nil {
		for _, channel := range notice.Channels {
			if channel == "station" {
				continue
			}
			if _, err := s.delivery.Enqueue(ctx, id, channel); err != nil {
				return fmt.Errorf("enqueue %s notification: %w", channel, err)
			}
		}
	}
	_ = s.publisher.Publish(ctx, recipients, messagingdomain.Event{
		Type: "notice", Title: notice.Title, Content: notice.Content, NoticeID: &id,
		Timestamp: time.Now().UTC().UnixMilli(),
	})
	return nil
}

func (s *Service) Withdraw(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Withdraw(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.Conflict("Only a published notice can be withdrawn.")
		}
		return err
	}
	return nil
}

func (s *Service) My(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]notificationdomain.Notice, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.My(ctx, userID, page, pageSize)
}

func (s *Service) MarkRead(ctx context.Context, noticeID, userID uuid.UUID) error {
	return s.repo.MarkRead(ctx, noticeID, userID)
}

func (s *Service) Unread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.Unread(ctx, userID)
}

func normalizeInput(input *notificationdomain.Input) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || len(input.Title) > 200 {
		return apperror.Validation("title", "Notice title is required and must not exceed 200 characters.")
	}
	if input.Content == "" || len(input.Content) > 100000 {
		return apperror.Validation("content", "Notice content is required and must not exceed 100000 characters.")
	}
	if input.NoticeType == 0 {
		input.NoticeType = 1
	}
	if input.NoticeType != 1 && input.NoticeType != 2 {
		return apperror.Validation("noticeType", "noticeType must be 1 or 2.")
	}
	if len(input.Channels) == 0 {
		input.Channels = []string{"station"}
	}
	for _, channel := range input.Channels {
		switch channel {
		case "station", "email", "sms":
		default:
			return apperror.Validation("channels", "Unsupported notification channel.")
		}
	}
	if input.TargetType == "" {
		input.TargetType = "all"
	}
	switch input.TargetType {
	case "all":
		input.TargetIDs = nil
	case "users", "departments":
		if len(input.TargetIDs) == 0 {
			return apperror.Validation("targetIds", "targetIds is required for the selected target type.")
		}
	default:
		return apperror.Validation("targetType", "targetType must be all, users, or departments.")
	}
	return nil
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
