package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"mars-admin-go/internal/modules/messaging/domain"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SendDirect(ctx context.Context, input domain.SendInput) (domain.Message, error) {
	if input.SenderID == uuid.Nil || input.ReceiverID == uuid.Nil || input.SenderID == input.ReceiverID {
		return domain.Message{}, apperror.Validation("receiverId", "A valid recipient is required.")
	}
	if err := normalizeMessage(&input); err != nil {
		return domain.Message{}, err
	}
	blocked, err := s.repo.IsBlocked(ctx, input.SenderID, input.ReceiverID)
	if err != nil {
		return domain.Message{}, err
	}
	if blocked {
		return domain.Message{}, apperror.Forbidden("Messaging is blocked between these users.")
	}
	message := domain.Message{
		ID:              mustID(),
		ClientMessageID: input.ClientMessageID,
		SenderID:        input.SenderID,
		ReceiverID:      &input.ReceiverID,
		MessageType:     input.MessageType,
		Content:         input.Content,
		Metadata:        input.Metadata,
		CreatedAt:       time.Now().UTC(),
	}
	created, err := s.repo.SendDirect(ctx, message)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Message{}, apperror.Conflict("This clientMessageId has already been used.")
		}
		return domain.Message{}, err
	}
	return created, nil
}

func (s *Service) DirectHistory(ctx context.Context, userID, peerID uuid.UUID, beforeSequence int64, limit int) ([]domain.Message, error) {
	if limit < 1 || limit > 100 {
		limit = 30
	}
	return s.repo.DirectHistory(ctx, userID, peerID, beforeSequence, limit)
}

func (s *Service) MarkDirectRead(ctx context.Context, userID, peerID uuid.UUID) error {
	return s.repo.MarkDirectRead(ctx, userID, peerID)
}

func (s *Service) Unread(ctx context.Context, userID uuid.UUID) (map[string]int64, error) {
	count, err := s.repo.UnreadDirect(ctx, userID)
	if err != nil {
		return nil, err
	}
	return map[string]int64{"chat": count}, nil
}

func (s *Service) Users(ctx context.Context, userID uuid.UUID, query string, limit int) ([]domain.ChatUser, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	return s.repo.ChatUsers(ctx, userID, strings.TrimSpace(query), limit)
}

func (s *Service) Block(ctx context.Context, userID, targetID uuid.UUID) error {
	if userID == targetID {
		return apperror.Validation("targetId", "You cannot block yourself.")
	}
	return s.repo.Block(ctx, userID, targetID)
}

func (s *Service) Unblock(ctx context.Context, userID, targetID uuid.UUID) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) Blocked(ctx context.Context, userID uuid.UUID) ([]domain.ChatUser, error) {
	return s.repo.BlockedUsers(ctx, userID)
}

func (s *Service) CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 1 || len(input.Name) > 150 {
		return domain.Group{}, apperror.Validation("name", "Group name must contain 1 to 150 characters.")
	}
	if input.OwnerID == uuid.Nil {
		return domain.Group{}, apperror.Unauthorized("Authentication is required.")
	}
	return s.repo.CreateGroup(ctx, input)
}

func (s *Service) Groups(ctx context.Context, userID uuid.UUID) ([]domain.Group, error) {
	return s.repo.Groups(ctx, userID)
}

func (s *Service) Group(ctx context.Context, groupID, userID uuid.UUID) (domain.Group, error) {
	if _, err := s.requireMember(ctx, groupID, userID); err != nil {
		return domain.Group{}, err
	}
	group, err := s.repo.Group(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Group{}, apperror.NotFound("Group")
	}
	return group, err
}

func (s *Service) UpdateGroup(ctx context.Context, groupID, userID uuid.UUID, name, avatarURL string) (domain.Group, error) {
	role, err := s.requireMember(ctx, groupID, userID)
	if err != nil {
		return domain.Group{}, err
	}
	if role != "owner" && role != "admin" {
		return domain.Group{}, apperror.Forbidden("Only a group owner or administrator can update the group.")
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return domain.Group{}, apperror.Validation("name", "Group name must contain 1 to 100 characters.")
	}
	return s.repo.UpdateGroup(ctx, groupID, name, avatarURL)
}

func (s *Service) Members(ctx context.Context, groupID, userID uuid.UUID) ([]domain.GroupMember, error) {
	if _, err := s.requireMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	return s.repo.Members(ctx, groupID)
}

func (s *Service) AddMembers(ctx context.Context, groupID, actorID uuid.UUID, userIDs []uuid.UUID) error {
	role, err := s.requireMember(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return apperror.Forbidden("Only group owners and administrators can add members.")
	}
	if len(userIDs) == 0 || len(userIDs) > 100 {
		return apperror.Validation("userIds", "Between 1 and 100 users are required.")
	}
	return s.repo.AddMembers(ctx, groupID, userIDs)
}

func (s *Service) RemoveMember(ctx context.Context, groupID, actorID, memberID uuid.UUID) error {
	actorRole, err := s.requireMember(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" && actorRole != "admin" && actorID != memberID {
		return apperror.Forbidden("Only group owners and administrators can remove members.")
	}
	return s.repo.RemoveMember(ctx, groupID, memberID)
}

func (s *Service) SetMemberRole(ctx context.Context, groupID, actorID, memberID uuid.UUID, role string) error {
	actorRole, err := s.requireMember(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" {
		return apperror.Forbidden("Only the group owner can change administrator roles.")
	}
	if role != "admin" && role != "member" {
		return apperror.Validation("role", "role must be admin or member.")
	}
	return s.repo.SetMemberRole(ctx, groupID, memberID, role)
}

func (s *Service) MuteMember(ctx context.Context, groupID, actorID, memberID uuid.UUID, muted bool) error {
	actorRole, err := s.requireMember(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" && actorRole != "admin" {
		return apperror.Forbidden("Only group owners and administrators can mute members.")
	}
	var until *time.Time
	if muted {
		value := time.Now().UTC().Add(24 * time.Hour)
		until = &value
	}
	return s.repo.MuteMember(ctx, groupID, memberID, until)
}

func (s *Service) TransferOwner(ctx context.Context, groupID, actorID, newOwnerID uuid.UUID) error {
	if newOwnerID == uuid.Nil || newOwnerID == actorID {
		return apperror.Validation("newOwnerId", "A different valid group member is required.")
	}
	return s.repo.TransferOwner(ctx, groupID, actorID, newOwnerID)
}

func (s *Service) QuitGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.repo.QuitGroup(ctx, groupID, userID)
}

func (s *Service) DissolveGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.repo.DissolveGroup(ctx, groupID, userID)
}

func (s *Service) SendGroup(ctx context.Context, groupID, senderID uuid.UUID, clientID, messageType, content string, metadata map[string]any) (domain.Message, error) {
	if _, err := s.repo.Group(ctx, groupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperror.Forbidden("The group is not active.")
		}
		return domain.Message{}, err
	}
	memberRole, err := s.requireMember(ctx, groupID, senderID)
	if err != nil {
		return domain.Message{}, err
	}
	input := domain.SendInput{
		SenderID: senderID, ClientMessageID: clientID, MessageType: messageType, Content: content, Metadata: metadata,
	}
	if err := normalizeMessage(&input); err != nil {
		return domain.Message{}, err
	}
	if memberRole == "member" {
		members, err := s.repo.Members(ctx, groupID)
		if err != nil {
			return domain.Message{}, err
		}
		for _, member := range members {
			if member.UserID == senderID && member.MutedUntil != nil && member.MutedUntil.After(time.Now().UTC()) {
				return domain.Message{}, apperror.Forbidden("You are muted in this group.")
			}
		}
	}
	message := domain.Message{
		ID: mustID(), ClientMessageID: clientID, SenderID: senderID, GroupID: &groupID,
		MessageType: input.MessageType, Content: input.Content, Metadata: metadata, CreatedAt: time.Now().UTC(),
	}
	created, err := s.repo.SendGroup(ctx, message)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Message{}, apperror.Conflict("This clientMessageId has already been used.")
		}
		return domain.Message{}, err
	}
	return created, nil
}

func (s *Service) GroupHistory(ctx context.Context, groupID, userID uuid.UUID, beforeSequence int64, limit int) ([]domain.Message, error) {
	if _, err := s.requireMember(ctx, groupID, userID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return s.repo.GroupHistory(ctx, groupID, beforeSequence, limit)
}

func (s *Service) requireMember(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	role, err := s.repo.MemberRole(ctx, groupID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.Forbidden("You are not a member of this group.")
	}
	return role, err
}

func normalizeMessage(input *domain.SendInput) error {
	input.Content = strings.TrimSpace(input.Content)
	length := utf8.RuneCountInString(input.Content)
	if length < 1 || length > 10000 {
		return apperror.Validation("content", "Message content must contain 1 to 10000 characters.")
	}
	if input.MessageType == "" {
		input.MessageType = "text"
	}
	switch input.MessageType {
	case "text", "image", "file":
	default:
		return apperror.Validation("messageType", "Unsupported message type.")
	}
	if len(input.ClientMessageID) > 120 {
		return apperror.Validation("clientMessageId", "clientMessageId must not exceed 120 characters.")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique")
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
