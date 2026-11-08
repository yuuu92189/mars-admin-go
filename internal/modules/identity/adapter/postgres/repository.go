package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	accesssqlc "mars-admin-go/internal/modules/access/adapter/postgres/sqlc"
	identitysqlc "mars-admin-go/internal/modules/identity/adapter/postgres/sqlc"
	"mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/modules/outbox/contracts"
	"mars-admin-go/internal/platform/apperror"
	platformoutbox "mars-admin-go/internal/platform/outbox"

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

func (r *Repository) FindUserByUsername(ctx context.Context, username string) (domain.User, error) {
	row, err := identitysqlc.New(r.pool).GetUserByUsername(ctx, username)
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *Repository) FindUserByPhone(ctx context.Context, phone string) (domain.User, error) {
	row, err := identitysqlc.New(r.pool).GetUserByPhone(ctx, pgText(phone))
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := identitysqlc.New(r.pool).GetUserByID(ctx, pgUUID(id))
	if err != nil {
		return domain.User{}, err
	}
	user := mapUser(row)
	org := organizationQueries{pool: r.pool}
	user.RoleIDs, _ = r.userRoleIDs(ctx, id)
	user.DepartmentIDs, _ = org.userDepartmentIDs(ctx, id)
	user.PositionIDs, _ = org.userPositionIDs(ctx, id)
	return user, nil
}

func (r *Repository) FindUserByIDInDataScope(ctx context.Context, targetID, actorID uuid.UUID, dataScope string) (domain.User, error) {
	row, err := identitysqlc.New(r.pool).GetUserByIDInDataScope(ctx, identitysqlc.GetUserByIDInDataScopeParams{
		ID: pgUUID(targetID), ID_2: pgUUID(actorID), Column3: dataScope,
	})
	if err != nil {
		return domain.User{}, err
	}
	user := mapUser(row)
	org := organizationQueries{pool: r.pool}
	user.RoleIDs, _ = r.userRoleIDs(ctx, targetID)
	user.DepartmentIDs, _ = org.userDepartmentIDs(ctx, targetID)
	user.PositionIDs, _ = org.userPositionIDs(ctx, targetID)
	return user, nil
}

func (r *Repository) CountDepartmentsWithinScope(ctx context.Context, actorID uuid.UUID, dataScope string, departmentIDs []uuid.UUID) (int64, error) {
	return identitysqlc.New(r.pool).CountDepartmentsWithinScope(ctx, identitysqlc.CountDepartmentsWithinScopeParams{
		UserID: pgUUID(actorID), Column2: dataScope, Column3: pgUUIDs(departmentIDs),
	})
}

func (r *Repository) CountPrivilegedRoles(ctx context.Context, roleIDs []uuid.UUID) (int64, error) {
	return identitysqlc.New(r.pool).CountPrivilegedRoles(ctx, pgUUIDs(roleIDs))
}

func (r *Repository) ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, int64, error) {
	q := identitysqlc.New(r.pool)
	offset := int32((filter.Page - 1) * filter.PageSize) // #nosec G115 -- application layer caps page size at 100.
	rows, err := q.ListUsers(ctx, identitysqlc.ListUsersParams{
		Column1: filter.Query,
		Column2: filter.Status,
		Column3: filter.UserType,
		Column4: filter.DataScope,
		ID:      pgUUID(filter.ActorID),
		Column6: filter.Sort,
		Limit:   int32(filter.PageSize), // #nosec G115 -- application layer caps page size at 100.
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountUsers(ctx, identitysqlc.CountUsersParams{
		Column1: filter.Query,
		Column2: filter.Status,
		Column3: filter.UserType,
		Column4: filter.DataScope,
		ID:      pgUUID(filter.ActorID),
	})
	if err != nil {
		return nil, 0, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapUser(row))
	}
	return users, total, nil
}

func (r *Repository) CreateUser(ctx context.Context, user domain.User, roleIDs, departmentIDs, positionIDs []uuid.UUID, primaryDepartmentID *uuid.UUID) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := identitysqlc.New(tx).CreateUser(ctx, identitysqlc.CreateUserParams{
		ID:           pgUUID(user.ID),
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Email:        pgText(user.Email),
		Phone:        pgText(user.Phone),
		AvatarUrl:    user.AvatarURL,
		UserType:     user.UserType,
		Status:       user.Status,
		Gender:       user.Gender,
	})
	if err != nil {
		return domain.User{}, mapPGError(err, "username")
	}
	identityQ := identitysqlc.New(tx)
	for _, roleID := range roleIDs {
		if err := identityQ.AddUserRole(ctx, identitysqlc.AddUserRoleParams{UserID: pgUUID(user.ID), RoleID: pgUUID(roleID)}); err != nil {
			return domain.User{}, err
		}
	}
	if err := assignOrganization(ctx, tx, user.ID, departmentIDs, primaryDepartmentID, positionIDs); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	created := mapUser(row)
	created.RoleIDs = roleIDs
	created.DepartmentIDs = departmentIDs
	created.PositionIDs = positionIDs
	return created, nil
}

func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput, _ string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := identitysqlc.New(tx)
	var previousStatus string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM identity_users
		WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&previousStatus); err != nil {
		return domain.User{}, err
	}
	row, err := q.UpdateUser(ctx, identitysqlc.UpdateUserParams{
		ID:          pgUUID(id),
		DisplayName: input.DisplayName,
		Email:       pgText(input.Email),
		Phone:       pgText(input.Phone),
		AvatarUrl:   input.AvatarURL,
		UserType:    input.UserType,
		Status:      input.Status,
		Gender:      input.Gender,
		Version:     input.Version,
	})
	if err != nil {
		return domain.User{}, err
	}
	if err := assignOrganization(ctx, tx, id, input.DepartmentIDs, input.PrimaryDepartmentID, input.PositionIDs); err != nil {
		return domain.User{}, err
	}
	if input.RoleIDs != nil {
		if err := q.SetUserRoles(ctx, pgUUID(id)); err != nil {
			return domain.User{}, err
		}
		for _, roleID := range input.RoleIDs {
			if err := q.AddUserRole(ctx, identitysqlc.AddUserRoleParams{
				UserID: pgUUID(id), RoleID: pgUUID(roleID),
			}); err != nil {
				return domain.User{}, err
			}
		}
	}
	if row.Status != "active" {
		if _, err := q.RevokeUserSessions(ctx, pgUUID(id)); err != nil {
			return domain.User{}, err
		}
	}
	if previousStatus != row.Status {
		if err := insertUserStatusEvent(ctx, tx, id, row.Status, row.Version); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *Repository) SoftDeleteUser(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := identitysqlc.New(tx)
	affected, err := q.SoftDeleteUser(ctx, pgUUID(id))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	if _, err := q.RevokeUserSessions(ctx, pgUUID(id)); err != nil {
		return err
	}
	if err := insertUserStatusEvent(ctx, tx, id, "deleted", time.Now().UTC().UnixNano()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	affected, err := identitysqlc.New(r.pool).UpdateUserPassword(ctx, identitysqlc.UpdateUserPasswordParams{ID: pgUUID(id), PasswordHash: hash})
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ChangePasswordAndRevokeSessions(ctx context.Context, id uuid.UUID, hash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := identitysqlc.New(tx)
	affected, err := q.UpdateUserPassword(ctx, identitysqlc.UpdateUserPasswordParams{ID: pgUUID(id), PasswordHash: hash})
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	if _, err := q.RevokeUserSessions(ctx, pgUUID(id)); err != nil {
		return err
	}
	if err := insertUserStatusEvent(ctx, tx, id, "password_changed", time.Now().UTC().UnixNano()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RecordFailedLogin(ctx context.Context, id uuid.UUID, maxAttempts int32, lockUntil time.Time) error {
	_, err := identitysqlc.New(r.pool).RecordFailedLogin(ctx, identitysqlc.RecordFailedLoginParams{
		ID:               pgUUID(id),
		FailedLoginCount: maxAttempts,
		LockedUntil:      pgTimestamp(lockUntil),
	})
	return err
}

func (r *Repository) ClearFailedLogin(ctx context.Context, id uuid.UUID) error {
	return identitysqlc.New(r.pool).ClearFailedLogin(ctx, pgUUID(id))
}

func (r *Repository) CreateSession(ctx context.Context, session domain.Session) error {
	familyID := session.RefreshFamilyID
	if familyID == uuid.Nil {
		familyID = session.ID
	}
	_, err := identitysqlc.New(r.pool).CreateSession(ctx, identitysqlc.CreateSessionParams{
		ID:                pgUUID(session.ID),
		UserID:            pgUUID(session.UserID),
		RefreshFamilyID:   pgUUID(familyID),
		RefreshGeneration: session.RefreshGeneration,
		ClientType:        session.ClientType,
		TokenHash:         session.TokenHash,
		RefreshTokenHash:  session.RefreshTokenHash,
		DeviceName:        session.DeviceName,
		IpAddress:         parseAddr(session.IPAddress),
		UserAgent:         session.UserAgent,
		ExpiresAt:         pgTimestamp(session.ExpiresAt),
		RefreshExpiresAt:  pgTimestamp(session.RefreshExpiresAt),
	})
	return err
}

func (r *Repository) GetSessionByAccessHash(ctx context.Context, hash []byte) (domain.Session, error) {
	row, err := identitysqlc.New(r.pool).GetSessionByAccessHash(ctx, hash)
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (r *Repository) GetSessionByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error) {
	row, err := identitysqlc.New(r.pool).GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		return domain.Session{}, err
	}
	return mapSession(row), nil
}

func (r *Repository) RotateSession(ctx context.Context, oldRefreshHash []byte, session domain.Session) error {
	_, err := identitysqlc.New(r.pool).RotateSessionTokens(ctx, identitysqlc.RotateSessionTokensParams{
		ID:                 pgUUID(session.ID),
		RefreshTokenHash:   oldRefreshHash,
		TokenHash:          session.TokenHash,
		RefreshTokenHash_2: session.RefreshTokenHash,
		ExpiresAt:          pgTimestamp(session.ExpiresAt),
		RefreshExpiresAt:   pgTimestamp(session.RefreshExpiresAt),
		ID_2:               pgUUID(mustID()),
	})
	return err
}

func (r *Repository) FindRefreshHistorySession(ctx context.Context, oldRefreshHash []byte) (uuid.UUID, error) {
	sessionID, err := identitysqlc.New(r.pool).FindRefreshHistorySession(ctx, oldRefreshHash)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.UUID(sessionID.Bytes), nil
}

func (r *Repository) ClassifyRefreshReplay(ctx context.Context, oldRefreshHash []byte, gracePeriod time.Duration) (domain.RefreshReplayResult, error) {
	row, err := identitysqlc.New(r.pool).ClassifyRefreshReplay(ctx, identitysqlc.ClassifyRefreshReplayParams{
		TokenHash: oldRefreshHash,
		Column2:   max(gracePeriod.Milliseconds(), 0),
	})
	if err != nil {
		return domain.RefreshReplayResult{}, err
	}
	result := domain.RefreshReplayResult{
		Found: row.SessionID.Valid, WithinGrace: row.WithinGrace, Revoked: row.Revoked,
	}
	if row.SessionID.Valid {
		result.SessionID = uuid.UUID(row.SessionID.Bytes)
	}
	return result, nil
}

func (r *Repository) CleanupRefreshHistory(ctx context.Context, retention time.Duration) (int64, error) {
	return identitysqlc.New(r.pool).DeleteExpiredRefreshHistory(ctx, max(retention.Milliseconds(), 0))
}

func (r *Repository) TouchSession(ctx context.Context, id uuid.UUID) error {
	return identitysqlc.New(r.pool).TouchSession(ctx, pgUUID(id))
}

func (r *Repository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	_, err := identitysqlc.New(r.pool).RevokeSession(ctx, pgUUID(id))
	return err
}

func (r *Repository) RevokeOwnedSession(ctx context.Context, id, userID uuid.UUID) error {
	affected, err := identitysqlc.New(r.pool).RevokeOwnedSession(ctx, identitysqlc.RevokeOwnedSessionParams{
		ID: pgUUID(id), UserID: pgUUID(userID),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := identitysqlc.New(r.pool).RevokeUserSessions(ctx, pgUUID(userID))
	return err
}

func (r *Repository) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	rows, err := identitysqlc.New(r.pool).ListUserSessions(ctx, pgUUID(userID))
	if err != nil {
		return nil, err
	}
	sessions := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, mapSession(row))
	}
	return sessions, nil
}

func (r *Repository) ListActiveSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, client_type, device_name, ip_address, user_agent,
		       expires_at, refresh_expires_at, last_seen_at, created_at,
		       revoked_at, token_hash, refresh_token_hash, refresh_family_id, refresh_generation
		FROM identity_sessions
		WHERE revoked_at IS NULL AND expires_at > now()
		ORDER BY last_seen_at DESC
		LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := make([]domain.Session, 0)
	for rows.Next() {
		var row identitysqlc.IdentitySession
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.ClientType, &row.DeviceName, &row.IpAddress, &row.UserAgent,
			&row.ExpiresAt, &row.RefreshExpiresAt, &row.LastSeenAt, &row.CreatedAt,
			&row.RevokedAt, &row.TokenHash, &row.RefreshTokenHash, &row.RefreshFamilyID, &row.RefreshGeneration,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, mapSession(row))
	}
	return sessions, rows.Err()
}

func (r *Repository) GetAccess(ctx context.Context, userID uuid.UUID) (domain.Access, error) {
	rows, err := accesssqlc.New(r.pool).GetUserAccess(ctx, pgUUID(userID))
	if err != nil {
		return domain.Access{}, err
	}
	roles := make([]string, 0)
	permissions := make([]string, 0)
	seenRoles := make(map[string]struct{})
	seenPermissions := make(map[string]struct{})
	access := domain.Access{DataScope: "self"}
	for _, row := range rows {
		if _, ok := seenRoles[row.RoleCode]; !ok {
			seenRoles[row.RoleCode] = struct{}{}
			roles = append(roles, row.RoleCode)
		}
		if row.PermissionCode != "" {
			if _, ok := seenPermissions[row.PermissionCode]; !ok {
				seenPermissions[row.PermissionCode] = struct{}{}
				permissions = append(permissions, row.PermissionCode)
			}
		}
		if row.IsSuperAdmin {
			access.IsSuperAdmin = true
			access.DataScope = "all"
		} else if access.DataScope == "self" && row.DataScope != "" {
			access.DataScope = row.DataScope
		}
	}
	access.Roles = roles
	access.Permissions = permissions
	return access, nil
}

func (r *Repository) AccessCacheEpoch(ctx context.Context) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT version FROM access_cache_epoch WHERE singleton = true`).Scan(&version)
	return version, err
}

func (r *Repository) SetUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := identitysqlc.New(tx)
	if _, err := tx.Exec(ctx, `DELETE FROM access_user_roles WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := q.AddUserRole(ctx, identitysqlc.AddUserRoleParams{UserID: pgUUID(userID), RoleID: pgUUID(roleID)}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) RecordLoginEvent(ctx context.Context, event domain.LoginEvent) error {
	var userID pgtype.UUID
	if event.UserID != nil {
		userID = pgUUID(*event.UserID)
	}
	return identitysqlc.New(r.pool).RecordLoginEvent(ctx, identitysqlc.RecordLoginEventParams{
		ID:            pgUUID(mustID()),
		UserID:        userID,
		Username:      event.Username,
		ClientType:    event.ClientType,
		Outcome:       event.Outcome,
		FailureReason: event.FailureReason,
		IpAddress:     parseAddr(event.IPAddress),
		UserAgent:     event.UserAgent,
	})
}

func (r *Repository) InsertOneTimeCode(ctx context.Context, purpose, subject string, codeHash []byte, expiresAt time.Time) error {
	return identitysqlc.New(r.pool).InsertOneTimeCode(ctx, identitysqlc.InsertOneTimeCodeParams{
		ID: pgUUID(mustID()), Purpose: purpose, Subject: subject,
		CodeHash: codeHash, ExpiresAt: pgTimestamp(expiresAt), MaxAttempts: 5,
	})
}

func (r *Repository) AttemptOneTimeCode(ctx context.Context, purpose, subject string, codeHash []byte) (domain.OneTimeCodeAttempt, error) {
	row, err := identitysqlc.New(r.pool).AttemptOneTimeCode(ctx, identitysqlc.AttemptOneTimeCodeParams{
		Purpose: purpose, Subject: subject, CodeHash: codeHash,
	})
	if err != nil {
		return domain.OneTimeCodeAttempt{}, err
	}
	return domain.OneTimeCodeAttempt{
		Valid: row.Valid, Attempts: row.Attempts, Consumed: row.Consumed,
	}, nil
}

func (r *Repository) InsertCaptcha(ctx context.Context, id uuid.UUID, kind string, answerHash []byte, targetX, tolerance int32, expiresAt time.Time) error {
	return identitysqlc.New(r.pool).InsertCaptchaChallenge(ctx, identitysqlc.InsertCaptchaChallengeParams{
		ID: pgUUID(id), Kind: kind, AnswerHash: answerHash,
		TargetX:   pgtype.Int4{Int32: targetX, Valid: kind == "slider"},
		Tolerance: tolerance, ExpiresAt: pgTimestamp(expiresAt),
	})
}

func (r *Repository) ConsumeCaptcha(ctx context.Context, id uuid.UUID, answerHash []byte, sliderX int32) (string, error) {
	return identitysqlc.New(r.pool).ConsumeCaptchaChallenge(ctx, identitysqlc.ConsumeCaptchaChallengeParams{
		ID: pgUUID(id), AnswerHash: answerHash, Column3: sliderX,
	})
}

func (r *Repository) SetMFASecret(ctx context.Context, userID uuid.UUID, encrypted []byte) error {
	affected, err := identitysqlc.New(r.pool).SetMFASecret(ctx, identitysqlc.SetMFASecretParams{
		ID: pgUUID(userID), MfaSecretEncrypted: encrypted,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) EnableMFA(ctx context.Context, userID uuid.UUID) error {
	affected, err := identitysqlc.New(r.pool).EnableMFA(ctx, pgUUID(userID))
	if err != nil {
		return err
	}
	if affected == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	_, err := identitysqlc.New(r.pool).DisableMFA(ctx, pgUUID(userID))
	return err
}

func (r *Repository) MFASecret(ctx context.Context, userID uuid.UUID) ([]byte, bool, error) {
	row, err := identitysqlc.New(r.pool).GetMFASecret(ctx, pgUUID(userID))
	if err != nil {
		return nil, false, err
	}
	return row.MfaSecretEncrypted, row.MfaEnabledAt.Valid, nil
}

func (r *Repository) FindSocialUser(ctx context.Context, provider, subject string) (domain.User, error) {
	row, err := identitysqlc.New(r.pool).GetSocialUser(ctx, identitysqlc.GetSocialUserParams{
		Provider: provider, Subject: subject,
	})
	if err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *Repository) CreateSocialUser(ctx context.Context, user domain.User, provider, subject string, metadata map[string]any) (domain.User, error) {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return domain.User{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := identitysqlc.New(tx)
	row, err := q.CreateUser(ctx, identitysqlc.CreateUserParams{
		ID:           pgUUID(user.ID),
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Email:        pgText(user.Email),
		Phone:        pgText(user.Phone),
		AvatarUrl:    user.AvatarURL,
		UserType:     user.UserType,
		Status:       user.Status,
		Gender:       user.Gender,
	})
	if err != nil {
		return domain.User{}, mapPGError(err, "social identity")
	}
	if err := q.CreateSocialIdentityStrict(ctx, identitysqlc.CreateSocialIdentityStrictParams{
		ID:       pgUUID(mustID()),
		UserID:   pgUUID(user.ID),
		Provider: provider,
		Subject:  subject,
		Metadata: encoded,
	}); err != nil {
		return domain.User{}, mapPGError(err, "social identity")
	}
	if err := q.DisablePasswordLogin(ctx, pgUUID(user.ID)); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return mapUser(row), nil
}

func (r *Repository) LinkSocialIdentity(ctx context.Context, userID uuid.UUID, provider, subject string, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return identitysqlc.New(r.pool).CreateSocialIdentity(ctx, identitysqlc.CreateSocialIdentityParams{
		ID: pgUUID(mustID()), UserID: pgUUID(userID), Provider: provider, Subject: subject, Metadata: encoded,
	})
}

func (r *Repository) UnlinkSocialIdentity(ctx context.Context, userID uuid.UUID, provider, subject string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var passwordLoginEnabled bool
	err = tx.QueryRow(ctx, `
		SELECT password_login_enabled
		FROM identity_users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, userID).Scan(&passwordLoginEnabled)
	if err != nil {
		return err
	}
	var identityCount int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM identity_social_identities
		WHERE user_id = $1`, userID).Scan(&identityCount); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		DELETE FROM identity_social_identities
		WHERE user_id = $1 AND provider = $2 AND subject = $3`, userID, provider, subject)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if !passwordLoginEnabled && identityCount <= 1 {
		return apperror.Conflict("The last authentication method cannot be removed.")
	}
	return tx.Commit(ctx)
}

func (r *Repository) FindRoleByCode(ctx context.Context, code string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM access_roles WHERE code = $1 AND deleted_at IS NULL`, code).Scan(&id)
	return id, err
}

func (r *Repository) EnsureBootstrapAdmin(ctx context.Context, username, passwordHash, email string) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID uuid.UUID
	var roleID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM identity_users WHERE username = $1 AND deleted_at IS NULL`, username).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		userID = mustID()
		_, err = tx.Exec(ctx, `
			INSERT INTO identity_users (id, username, password_hash, display_name, email, user_type, status)
			VALUES ($1, $2, $3, $4, $5, 'admin', 'active')`,
			userID, username, passwordHash, username, email,
		)
	} else if err == nil {
		_, err = tx.Exec(ctx, `UPDATE identity_users SET password_hash = $2, updated_at = now(), status = 'active' WHERE id = $1`, userID, passwordHash)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.QueryRow(ctx, `SELECT id FROM access_roles WHERE code = 'admin'`).Scan(&roleID); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO access_user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, roleID); err != nil {
		return uuid.Nil, err
	}
	return userID, tx.Commit(ctx)
}

type organizationQueries struct {
	pool *pgxpool.Pool
}

func (q organizationQueries) userDepartmentIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, `SELECT department_id FROM organization_memberships WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUUIDs(rows)
}

func (q organizationQueries) userPositionIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, `SELECT position_id FROM organization_user_positions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUUIDs(rows)
}

func (r *Repository) userRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT role_id FROM access_user_roles WHERE user_id = $1 ORDER BY role_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUUIDs(rows)
}

func assignOrganization(ctx context.Context, tx pgx.Tx, userID uuid.UUID, departmentIDs []uuid.UUID, primary *uuid.UUID, positionIDs []uuid.UUID) error {
	if departmentIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM organization_memberships WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for _, departmentID := range departmentIDs {
			isPrimary := primary != nil && *primary == departmentID
			if _, err := tx.Exec(ctx, `
				INSERT INTO organization_memberships (user_id, department_id, is_primary)
				VALUES ($1, $2, $3) ON CONFLICT (user_id, department_id) DO UPDATE SET is_primary = excluded.is_primary`,
				userID, departmentID, isPrimary,
			); err != nil {
				return err
			}
		}
	}
	if positionIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM organization_user_positions WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for _, positionID := range positionIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO organization_user_positions (user_id, position_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, positionID); err != nil {
				return err
			}
		}
	}
	return nil
}

func mapUser(row identitysqlc.IdentityUser) domain.User {
	return domain.User{
		ID:                   uuid.UUID(row.ID.Bytes),
		Username:             row.Username,
		DisplayName:          row.DisplayName,
		Email:                textValue(row.Email),
		Phone:                textValue(row.Phone),
		AvatarURL:            row.AvatarUrl,
		UserType:             row.UserType,
		Status:               row.Status,
		Gender:               row.Gender,
		LastLoginAt:          timePtr(row.LastLoginAt),
		CreatedAt:            timeValue(row.CreatedAt),
		UpdatedAt:            timeValue(row.UpdatedAt),
		Version:              row.Version,
		MFAEnabled:           row.MfaEnabledAt.Valid,
		PasswordHash:         row.PasswordHash,
		PasswordLoginEnabled: row.PasswordLoginEnabled,
		FailedLogin:          row.FailedLoginCount,
		LockedUntil:          timePtr(row.LockedUntil),
	}
}

func mapSession(row identitysqlc.IdentitySession) domain.Session {
	return domain.Session{
		ID:                uuid.UUID(row.ID.Bytes),
		UserID:            uuid.UUID(row.UserID.Bytes),
		RefreshFamilyID:   uuid.UUID(row.RefreshFamilyID.Bytes),
		RefreshGeneration: row.RefreshGeneration,
		ClientType:        row.ClientType,
		DeviceName:        row.DeviceName,
		IPAddress:         addrString(row.IpAddress),
		UserAgent:         row.UserAgent,
		ExpiresAt:         timeValue(row.ExpiresAt),
		RefreshExpiresAt:  timeValue(row.RefreshExpiresAt),
		LastSeenAt:        timeValue(row.LastSeenAt),
		CreatedAt:         timeValue(row.CreatedAt),
		TokenHash:         row.TokenHash,
		RefreshTokenHash:  row.RefreshTokenHash,
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func pgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	values := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		values = append(values, pgUUID(id))
	}
	return values
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func parseAddr(value string) *netip.Addr {
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return nil
	}
	return &addr
}

func addrString(value *netip.Addr) string {
	if value == nil {
		return ""
	}
	return value.String()
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

func mapPGError(err error, field string) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "duplicate") || strings.Contains(message, "unique") {
		return apperror.Conflict(fmt.Sprintf("%s is already in use.", field))
	}
	return err
}

func insertUserStatusEvent(ctx context.Context, tx pgx.Tx, userID uuid.UUID, status string, version int64) error {
	event, err := platformoutbox.NewEvent(
		"identity_user", userID.String(), contracts.EventUserStatusChanged,
		fmt.Sprintf("version:%d", version),
		contracts.UserStatusChanged{UserID: userID, Status: status, ChangedAt: time.Now().UTC()},
	)
	if err != nil {
		return err
	}
	return platformoutbox.Insert(ctx, tx, event)
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
