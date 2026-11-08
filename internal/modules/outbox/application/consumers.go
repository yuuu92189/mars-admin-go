package application

import (
	"context"
	"encoding/json"
	"fmt"

	messagingdomain "mars-admin-go/internal/modules/messaging/domain"
	"mars-admin-go/internal/modules/outbox/contracts"
	storagedomain "mars-admin-go/internal/modules/storage/domain"
	platformoutbox "mars-admin-go/internal/platform/outbox"

	"github.com/google/uuid"
)

type DeliveryEnqueuer interface {
	Enqueue(ctx context.Context, noticeID uuid.UUID, channel string) (int64, error)
}

type Consumers struct {
	realtime messagingdomain.Publisher
	delivery DeliveryEnqueuer
	storage  storagedomain.Backend
}

func NewConsumers(realtime messagingdomain.Publisher, delivery DeliveryEnqueuer, storage ...storagedomain.Backend) *Consumers {
	consumer := &Consumers{realtime: realtime, delivery: delivery}
	if len(storage) > 0 {
		consumer.storage = storage[0]
	}
	return consumer
}

func (c *Consumers) HandleRealtime(ctx context.Context, event platformoutbox.Event) error {
	var payload contracts.RealtimeEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode realtime outbox payload: %w", err)
	}
	if payload.Event.ID == "" {
		payload.Event.ID = event.ID.String()
	}
	return c.realtime.Publish(ctx, payload.Recipients, payload.Event)
}

func (c *Consumers) HandleNoticePublished(ctx context.Context, event platformoutbox.Event) error {
	var payload contracts.NoticePublished
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode notice outbox payload: %w", err)
	}
	if c.delivery != nil {
		for _, channel := range payload.Channels {
			if channel == "station" {
				continue
			}
			if _, err := c.delivery.Enqueue(ctx, payload.NoticeID, channel); err != nil {
				return fmt.Errorf("enqueue %s notice delivery: %w", channel, err)
			}
		}
	}
	return c.realtime.Publish(ctx, payload.Recipients, messagingdomain.Event{
		ID: event.ID.String(), Type: "notice", Title: payload.Title, Content: payload.Content,
		NoticeID: &payload.NoticeID, Timestamp: payload.PublishedAt.UnixMilli(),
	})
}

func (c *Consumers) HandleUserStatusChanged(ctx context.Context, event platformoutbox.Event) error {
	var payload contracts.UserStatusChanged
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode user status outbox payload: %w", err)
	}
	return c.realtime.Publish(ctx, []uuid.UUID{payload.UserID}, messagingdomain.Event{
		ID: event.ID.String(), Type: "userStatus", UserID: &payload.UserID,
		Content: payload.Status, Timestamp: payload.ChangedAt.UnixMilli(),
	})
}

func (c *Consumers) HandleWorkflowDecision(ctx context.Context, event platformoutbox.Event) error {
	var payload contracts.WorkflowDecision
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode workflow outbox payload: %w", err)
	}
	return c.realtime.Publish(ctx, payload.Recipients, messagingdomain.Event{
		ID: event.ID.String(), Type: "workflow", Content: payload.Action + ":" + payload.Status,
		Timestamp: payload.DecidedAt.UnixMilli(),
	})
}

func (c *Consumers) HandleFileDeleted(ctx context.Context, event platformoutbox.Event) error {
	var payload contracts.FileDeleted
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode file deletion outbox payload: %w", err)
	}
	if c.storage == nil {
		return fmt.Errorf("storage backend is not configured for provider %q", payload.StorageProvider)
	}
	if payload.StorageProvider != "" && c.storage.Name() != payload.StorageProvider {
		return fmt.Errorf("storage provider mismatch: event=%s worker=%s", payload.StorageProvider, c.storage.Name())
	}
	return c.storage.Delete(ctx, payload.StorageKey)
}
