package contracts

import (
	"time"

	messagingdomain "mars-admin-go/internal/modules/messaging/domain"

	"github.com/google/uuid"
)

const (
	EventUserStatusChanged = "identity.user.status_changed"
	EventNoticePublished   = "notification.notice.published"
	EventDirectMessage     = "messaging.direct_message.created"
	EventGroupMessage      = "messaging.group_message.created"
	EventWorkflowDecision  = "workflow.task.decided"
	EventFileDeleted       = "storage.file.deleted"
)

type RealtimeEvent struct {
	Recipients []uuid.UUID           `json:"recipients"`
	Event      messagingdomain.Event `json:"event"`
}

type NoticePublished struct {
	NoticeID    uuid.UUID   `json:"noticeId"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Channels    []string    `json:"channels"`
	Recipients  []uuid.UUID `json:"recipients"`
	PublishedAt time.Time   `json:"publishedAt"`
}

type UserStatusChanged struct {
	UserID    uuid.UUID `json:"userId"`
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changedAt"`
}

type WorkflowDecision struct {
	InstanceID uuid.UUID   `json:"instanceId"`
	TaskID     uuid.UUID   `json:"taskId"`
	Action     string      `json:"action"`
	NextNode   string      `json:"nextNode,omitempty"`
	Status     string      `json:"status"`
	Recipients []uuid.UUID `json:"recipients"`
	DecidedAt  time.Time   `json:"decidedAt"`
}

type FileDeleted struct {
	FileID          uuid.UUID `json:"fileId"`
	StorageKey      string    `json:"storageKey"`
	StorageProvider string    `json:"storageProvider"`
	DeletedAt       time.Time `json:"deletedAt"`
}
