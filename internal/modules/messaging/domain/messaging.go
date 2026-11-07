package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID              uuid.UUID      `json:"id"`
	ClientMessageID string         `json:"clientMessageId,omitempty"`
	SenderID        uuid.UUID      `json:"senderId"`
	SenderName      string         `json:"senderName,omitempty"`
	SenderAvatar    string         `json:"senderAvatar,omitempty"`
	ReceiverID      *uuid.UUID     `json:"receiverId,omitempty"`
	GroupID         *uuid.UUID     `json:"groupId,omitempty"`
	MessageType     string         `json:"messageType"`
	Content         string         `json:"content"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Sequence        int64          `json:"sequence"`
	Read            bool           `json:"read"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type SendInput struct {
	SenderID        uuid.UUID      `json:"-"`
	ReceiverID      uuid.UUID      `json:"receiverId"`
	ClientMessageID string         `json:"clientMessageId"`
	MessageType     string         `json:"messageType"`
	Content         string         `json:"content"`
	Metadata        map[string]any `json:"metadata"`
}

type Group struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarUrl"`
	OwnerID   uuid.UUID `json:"ownerId"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GroupMember struct {
	GroupID     uuid.UUID  `json:"groupId"`
	UserID      uuid.UUID  `json:"userId"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	AvatarURL   string     `json:"avatarUrl"`
	Role        string     `json:"role"`
	MutedUntil  *time.Time `json:"mutedUntil,omitempty"`
	JoinedAt    time.Time  `json:"joinedAt"`
}

type ChatUser struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl"`
	Online      bool      `json:"online"`
}

type CreateGroupInput struct {
	OwnerID   uuid.UUID   `json:"-"`
	Name      string      `json:"name"`
	AvatarURL string      `json:"avatarUrl"`
	MemberIDs []uuid.UUID `json:"memberIds"`
}

type Repository interface {
	SendDirect(ctx context.Context, message Message) (Message, error)
	DirectHistory(ctx context.Context, userID, peerID uuid.UUID, beforeSequence int64, limit int) ([]Message, error)
	UnreadDirect(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkDirectRead(ctx context.Context, userID, peerID uuid.UUID) error
	ChatUsers(ctx context.Context, userID uuid.UUID, query string, limit int) ([]ChatUser, error)
	IsBlocked(ctx context.Context, first, second uuid.UUID) (bool, error)
	Block(ctx context.Context, userID, targetID uuid.UUID) error
	Unblock(ctx context.Context, userID, targetID uuid.UUID) error
	BlockedUsers(ctx context.Context, userID uuid.UUID) ([]ChatUser, error)
	CreateGroup(ctx context.Context, input CreateGroupInput) (Group, error)
	Groups(ctx context.Context, userID uuid.UUID) ([]Group, error)
	Group(ctx context.Context, id uuid.UUID) (Group, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, name, avatarURL string) (Group, error)
	Members(ctx context.Context, groupID uuid.UUID) ([]GroupMember, error)
	MemberRole(ctx context.Context, groupID, userID uuid.UUID) (string, error)
	AddMembers(ctx context.Context, groupID uuid.UUID, userIDs []uuid.UUID) error
	RemoveMember(ctx context.Context, groupID, memberID, ownerID uuid.UUID) error
	SetMemberRole(ctx context.Context, groupID, memberID uuid.UUID, role string) error
	MuteMember(ctx context.Context, groupID, memberID uuid.UUID, until *time.Time) error
	TransferOwner(ctx context.Context, groupID, currentOwnerID, newOwnerID uuid.UUID) error
	QuitGroup(ctx context.Context, groupID, userID uuid.UUID) error
	DissolveGroup(ctx context.Context, groupID, ownerID uuid.UUID) error
	SendGroup(ctx context.Context, message Message) (Message, error)
	GroupHistory(ctx context.Context, groupID uuid.UUID, beforeSequence int64, limit int) ([]Message, error)
}

type Event struct {
	Type      string           `json:"type"`
	Message   *Message         `json:"message,omitempty"`
	Title     string           `json:"title,omitempty"`
	Content   string           `json:"content,omitempty"`
	UserID    *uuid.UUID       `json:"userId,omitempty"`
	NoticeID  *uuid.UUID       `json:"noticeId,omitempty"`
	Unread    map[string]int64 `json:"unread,omitempty"`
	Timestamp int64            `json:"time,omitempty"`
}

type Publisher interface {
	Publish(ctx context.Context, recipients []uuid.UUID, event Event) error
}

type PublisherFunc func(ctx context.Context, recipients []uuid.UUID, event Event) error

func (f PublisherFunc) Publish(ctx context.Context, recipients []uuid.UUID, event Event) error {
	return f(ctx, recipients, event)
}
