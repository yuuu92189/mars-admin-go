package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sqlc "mars-admin-go/internal/modules/messaging/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/messaging/domain"
	"mars-admin-go/internal/modules/outbox/contracts"
	platformoutbox "mars-admin-go/internal/platform/outbox"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SendDirect(ctx context.Context, message domain.Message) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	metadata, _ := json.Marshal(message.Metadata)
	row, err := sqlc.New(tx).InsertDirectMessage(ctx, sqlc.InsertDirectMessageParams{
		ID: pgUUID(message.ID), ClientMessageID: pgText(message.ClientMessageID),
		SenderID: pgUUID(message.SenderID), ReceiverID: pgUUID(*message.ReceiverID),
		MessageType: message.MessageType, Content: message.Content, Metadata: metadata,
	})
	if err != nil {
		return domain.Message{}, err
	}
	created := mapMessage(row)
	event, err := platformoutbox.NewEvent(
		"messaging_message", created.ID.String(), contracts.EventDirectMessage, "created",
		contracts.RealtimeEvent{
			Recipients: []uuid.UUID{created.SenderID, *created.ReceiverID},
			Event: domain.Event{
				Type: "chat", Message: &created, Timestamp: created.CreatedAt.UnixMilli(),
			},
		},
	)
	if err != nil {
		return domain.Message{}, err
	}
	if err := platformoutbox.Insert(ctx, tx, event); err != nil {
		return domain.Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, err
	}
	return created, nil
}

func (r *Repository) DirectHistory(ctx context.Context, userID, peerID uuid.UUID, beforeSequence int64, limit int) ([]domain.Message, error) {
	rows, err := sqlc.New(r.pool).ListDirectMessages(ctx, sqlc.ListDirectMessagesParams{
		SenderID: pgUUID(userID), ReceiverID: pgUUID(peerID),
		Column3: beforeSequence, Limit: int32(limit), // #nosec G115 -- application layer caps limit at 100.
	})
	if err != nil {
		return nil, err
	}
	return mapMessages(rows), nil
}

func (r *Repository) UnreadDirect(ctx context.Context, userID uuid.UUID) (int64, error) {
	return sqlc.New(r.pool).CountDirectUnread(ctx, pgUUID(userID))
}

func (r *Repository) MarkDirectRead(ctx context.Context, userID, peerID uuid.UUID) error {
	return sqlc.New(r.pool).MarkDirectMessagesRead(ctx, sqlc.MarkDirectMessagesReadParams{
		UserID: pgUUID(userID), SenderID: pgUUID(peerID), CreatedAt: pgTimestamp(time.Now().UTC()),
	})
}

func (r *Repository) ChatUsers(ctx context.Context, userID uuid.UUID, query string, limit int) ([]domain.ChatUser, error) {
	rows, err := sqlc.New(r.pool).ListChatUsers(ctx, sqlc.ListChatUsersParams{
		ID: pgUUID(userID), Column2: query, Limit: int32(limit), // #nosec G115 -- application layer caps limit at 200.
	})
	if err != nil {
		return nil, err
	}
	users := make([]domain.ChatUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, domain.ChatUser{
			ID: uuid.UUID(row.ID.Bytes), Username: row.Username, DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl,
		})
	}
	return users, nil
}

func (r *Repository) IsBlocked(ctx context.Context, first, second uuid.UUID) (bool, error) {
	count, err := sqlc.New(r.pool).CountBlockedConversation(ctx, sqlc.CountBlockedConversationParams{
		UserID: pgUUID(first), BlockedUserID: pgUUID(second),
	})
	return count > 0, err
}

func (r *Repository) Block(ctx context.Context, userID, targetID uuid.UUID) error {
	return sqlc.New(r.pool).BlockUser(ctx, sqlc.BlockUserParams{UserID: pgUUID(userID), BlockedUserID: pgUUID(targetID)})
}

func (r *Repository) Unblock(ctx context.Context, userID, targetID uuid.UUID) error {
	_, err := sqlc.New(r.pool).UnblockUser(ctx, sqlc.UnblockUserParams{UserID: pgUUID(userID), BlockedUserID: pgUUID(targetID)})
	return err
}

func (r *Repository) BlockedUsers(ctx context.Context, userID uuid.UUID) ([]domain.ChatUser, error) {
	rows, err := sqlc.New(r.pool).ListBlockedUsers(ctx, pgUUID(userID))
	if err != nil {
		return nil, err
	}
	users := make([]domain.ChatUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, domain.ChatUser{
			ID: uuid.UUID(row.ID.Bytes), Username: row.Username, DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl,
		})
	}
	return users, nil
}

func (r *Repository) CreateGroup(ctx context.Context, input domain.CreateGroupInput) (domain.Group, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Group{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := sqlc.New(tx).CreateGroup(ctx, sqlc.CreateGroupParams{
		ID: pgUUID(mustID()), Name: input.Name, AvatarUrl: input.AvatarURL, OwnerID: pgUUID(input.OwnerID),
	})
	if err != nil {
		return domain.Group{}, err
	}
	q := sqlc.New(tx)
	if err := q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{GroupID: row.ID, UserID: pgUUID(input.OwnerID), MemberRole: "owner"}); err != nil {
		return domain.Group{}, err
	}
	for _, memberID := range input.MemberIDs {
		if memberID == input.OwnerID || memberID == uuid.Nil {
			continue
		}
		if err := q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{GroupID: row.ID, UserID: pgUUID(memberID), MemberRole: "member"}); err != nil {
			return domain.Group{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Group{}, err
	}
	return mapGroup(row), nil
}

func (r *Repository) Groups(ctx context.Context, userID uuid.UUID) ([]domain.Group, error) {
	rows, err := sqlc.New(r.pool).ListGroupsForUser(ctx, pgUUID(userID))
	if err != nil {
		return nil, err
	}
	groups := make([]domain.Group, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, mapGroup(row))
	}
	return groups, nil
}

func (r *Repository) Group(ctx context.Context, id uuid.UUID) (domain.Group, error) {
	row, err := sqlc.New(r.pool).GetGroup(ctx, pgUUID(id))
	if err != nil {
		return domain.Group{}, err
	}
	return mapGroup(row), nil
}

func (r *Repository) UpdateGroup(ctx context.Context, id uuid.UUID, name, avatarURL string) (domain.Group, error) {
	row, err := sqlc.New(r.pool).UpdateGroup(ctx, sqlc.UpdateGroupParams{
		ID: pgUUID(id), Name: name, AvatarUrl: avatarURL,
	})
	if err != nil {
		return domain.Group{}, err
	}
	return mapGroup(row), nil
}

func (r *Repository) Members(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error) {
	rows, err := sqlc.New(r.pool).ListGroupMembers(ctx, pgUUID(groupID))
	if err != nil {
		return nil, err
	}
	members := make([]domain.GroupMember, 0, len(rows))
	for _, row := range rows {
		member := domain.GroupMember{
			GroupID: uuid.UUID(row.GroupID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
			Username: row.Username, DisplayName: row.DisplayName, AvatarURL: row.AvatarUrl,
			Role: row.MemberRole, JoinedAt: row.JoinedAt.Time.UTC(),
		}
		if row.MutedUntil.Valid {
			value := row.MutedUntil.Time.UTC()
			member.MutedUntil = &value
		}
		members = append(members, member)
	}
	return members, nil
}

func (r *Repository) MemberRole(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx, `
		SELECT member_role FROM messaging_group_members
		WHERE group_id = $1 AND user_id = $2 AND left_at IS NULL`, groupID, userID).Scan(&role)
	return role, err
}

func (r *Repository) AddMembers(ctx context.Context, groupID uuid.UUID, userIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	for _, userID := range userIDs {
		if err := q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{GroupID: pgUUID(groupID), UserID: pgUUID(userID), MemberRole: "member"}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) RemoveMember(ctx context.Context, groupID, memberID, ownerID uuid.UUID) error {
	_, err := sqlc.New(r.pool).RemoveGroupMember(ctx, sqlc.RemoveGroupMemberParams{
		GroupID: pgUUID(groupID), UserID: pgUUID(memberID), UserID_2: pgUUID(ownerID),
	})
	return err
}

func (r *Repository) SetMemberRole(ctx context.Context, groupID, memberID uuid.UUID, role string) error {
	_, err := sqlc.New(r.pool).UpdateGroupMemberRole(ctx, sqlc.UpdateGroupMemberRoleParams{
		GroupID: pgUUID(groupID), UserID: pgUUID(memberID), MemberRole: role,
	})
	return err
}

func (r *Repository) MuteMember(ctx context.Context, groupID, memberID uuid.UUID, until *time.Time) error {
	_, err := sqlc.New(r.pool).SetGroupMemberMuted(ctx, sqlc.SetGroupMemberMutedParams{
		GroupID: pgUUID(groupID), UserID: pgUUID(memberID), MutedUntil: optionalTimestamp(until),
	})
	return err
}

func (r *Repository) TransferOwner(ctx context.Context, groupID, currentOwnerID, newOwnerID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	group, err := q.GetGroup(ctx, pgUUID(groupID))
	if err != nil {
		return err
	}
	if uuid.UUID(group.OwnerID.Bytes) != currentOwnerID {
		return errors.New("only the group owner can transfer ownership")
	}
	if err := q.TransferGroupMemberRoles(ctx, sqlc.TransferGroupMemberRolesParams{
		GroupID: pgUUID(groupID), UserID: pgUUID(newOwnerID), UserID_2: pgUUID(currentOwnerID),
	}); err != nil {
		return err
	}
	if err := q.TransferGroupOwner(ctx, sqlc.TransferGroupOwnerParams{ID: pgUUID(groupID), OwnerID: pgUUID(newOwnerID)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) QuitGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	_, err := sqlc.New(r.pool).QuitGroup(ctx, sqlc.QuitGroupParams{GroupID: pgUUID(groupID), UserID: pgUUID(userID)})
	return err
}

func (r *Repository) DissolveGroup(ctx context.Context, groupID, ownerID uuid.UUID) error {
	_, err := sqlc.New(r.pool).DissolveGroup(ctx, sqlc.DissolveGroupParams{ID: pgUUID(groupID), OwnerID: pgUUID(ownerID)})
	return err
}

func (r *Repository) SendGroup(ctx context.Context, message domain.Message) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	metadata, _ := json.Marshal(message.Metadata)
	row, err := q.InsertGroupMessage(ctx, sqlc.InsertGroupMessageParams{
		ID: pgUUID(message.ID), ClientMessageID: pgText(message.ClientMessageID),
		SenderID: pgUUID(message.SenderID), GroupID: pgUUID(*message.GroupID),
		MessageType: message.MessageType, Content: message.Content, Metadata: metadata,
	})
	if err != nil {
		return domain.Message{}, err
	}
	created := mapMessage(row)
	members, err := q.ListGroupMembers(ctx, pgUUID(*created.GroupID))
	if err != nil {
		return domain.Message{}, err
	}
	recipients := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		recipients = append(recipients, uuid.UUID(member.UserID.Bytes))
	}
	event, err := platformoutbox.NewEvent(
		"messaging_message", created.ID.String(), contracts.EventGroupMessage, "created",
		contracts.RealtimeEvent{
			Recipients: recipients,
			Event: domain.Event{
				Type: "groupChat", Message: &created, Timestamp: created.CreatedAt.UnixMilli(),
			},
		},
	)
	if err != nil {
		return domain.Message{}, err
	}
	if err := platformoutbox.Insert(ctx, tx, event); err != nil {
		return domain.Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, err
	}
	return created, nil
}

func (r *Repository) GroupHistory(ctx context.Context, groupID uuid.UUID, beforeSequence int64, limit int) ([]domain.Message, error) {
	rows, err := sqlc.New(r.pool).ListGroupMessages(ctx, sqlc.ListGroupMessagesParams{
		GroupID: pgUUID(groupID), Column2: beforeSequence, Limit: int32(limit), // #nosec G115 -- application layer caps limit at 100.
	})
	if err != nil {
		return nil, err
	}
	return mapMessages(rows), nil
}

func mapMessages(rows []sqlc.MessagingMessage) []domain.Message {
	messages := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, mapMessage(row))
	}
	return messages
}

func mapMessage(row sqlc.MessagingMessage) domain.Message {
	message := domain.Message{
		ID: uuid.UUID(row.ID.Bytes), ClientMessageID: text(row.ClientMessageID),
		SenderID: uuid.UUID(row.SenderID.Bytes), MessageType: row.MessageType,
		Content: row.Content, Sequence: row.SequenceNumber.Int64, CreatedAt: row.CreatedAt.Time.UTC(),
	}
	_ = json.Unmarshal(row.Metadata, &message.Metadata)
	if row.ReceiverID.Valid {
		id := uuid.UUID(row.ReceiverID.Bytes)
		message.ReceiverID = &id
	}
	if row.GroupID.Valid {
		id := uuid.UUID(row.GroupID.Bytes)
		message.GroupID = &id
	}
	return message
}

func mapGroup(row sqlc.MessagingGroup) domain.Group {
	return domain.Group{
		ID: uuid.UUID(row.ID.Bytes), Name: row.Name, AvatarURL: row.AvatarUrl,
		OwnerID: uuid.UUID(row.OwnerID.Bytes), Status: row.Status,
		CreatedAt: row.CreatedAt.Time.UTC(), UpdatedAt: row.UpdatedAt.Time.UTC(),
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func text(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func optionalTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgTimestamp(*value)
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
