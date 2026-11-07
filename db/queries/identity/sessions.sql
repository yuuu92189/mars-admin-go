-- name: CreateSession :one
INSERT INTO identity_sessions (
    id, user_id, refresh_family_id, refresh_generation, client_type, token_hash, refresh_token_hash, device_name,
    ip_address, user_agent, expires_at, refresh_expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetSessionByAccessHash :one
SELECT *
FROM identity_sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: GetSessionByRefreshHash :one
SELECT *
FROM identity_sessions
WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND refresh_expires_at > now();

-- name: RotateSessionTokens :one
WITH rotated AS (
    UPDATE identity_sessions AS session
    SET token_hash = $3,
        refresh_token_hash = $4,
        expires_at = $5,
        refresh_expires_at = $6,
        refresh_generation = refresh_generation + 1,
        last_seen_at = now()
    WHERE session.id = $1
      AND session.refresh_token_hash = $2
      AND session.revoked_at IS NULL
      AND session.refresh_expires_at > now()
      AND EXISTS (
          SELECT 1
          FROM identity_users AS account
          WHERE account.id = session.user_id
            AND account.status = 'active'
            AND account.deleted_at IS NULL
      )
    RETURNING *
), history AS (
    INSERT INTO identity_refresh_token_history (
        id, session_id, token_hash, generation
    )
    SELECT $7, rotated.id, $2, rotated.refresh_generation
    FROM rotated
)
SELECT * FROM rotated;

-- name: FindRefreshHistorySession :one
SELECT session_id
FROM identity_refresh_token_history
WHERE token_hash = $1;

-- name: RevokeRefreshReplay :one
WITH replayed AS (
    SELECT history.session_id
    FROM identity_refresh_token_history AS history
    WHERE history.token_hash = $1
), revoked AS (
    UPDATE identity_sessions AS session
    SET revoked_at = now()
    FROM replayed
    WHERE session.id = replayed.session_id
      AND session.revoked_at IS NULL
    RETURNING session.id
)
SELECT EXISTS (SELECT 1 FROM replayed)::boolean AS replayed;

-- name: TouchSession :exec
UPDATE identity_sessions SET last_seen_at = now() WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeSession :execrows
UPDATE identity_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeOwnedSession :execrows
UPDATE identity_sessions
SET revoked_at = now()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: RevokeUserSessions :execrows
UPDATE identity_sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: ListUserSessions :many
SELECT *
FROM identity_sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
ORDER BY last_seen_at DESC;

-- name: RecordLoginEvent :exec
INSERT INTO audit_login_events (
    id, user_id, username, client_type, outcome, failure_reason, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: InsertOneTimeCode :exec
WITH invalidated AS (
    UPDATE identity_one_time_codes AS code
    SET consumed_at = now()
    WHERE code.purpose = $2 AND code.subject = $3
      AND code.consumed_at IS NULL AND code.expires_at > now()
)
INSERT INTO identity_one_time_codes (
    id, purpose, subject, code_hash, expires_at, max_attempts
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: AttemptOneTimeCode :one
WITH candidate AS (
    SELECT
        code.id,
        (code.code_hash = $3)::boolean AS matches,
        code.max_attempts,
        code.attempts,
        code.consumed_at,
        (
            code.consumed_at IS NULL
            AND code.expires_at > now()
            AND code.attempts < code.max_attempts
        )::boolean AS can_attempt
    FROM identity_one_time_codes AS code
    WHERE code.purpose = $1 AND code.subject = $2
    ORDER BY code.created_at DESC
    LIMIT 1
    FOR UPDATE
), updated AS (
    UPDATE identity_one_time_codes AS code
    SET attempts = code.attempts + 1,
        consumed_at = CASE
            WHEN candidate.matches OR code.attempts + 1 >= candidate.max_attempts
            THEN now()
            ELSE code.consumed_at
        END
    FROM candidate
    WHERE code.id = candidate.id AND candidate.can_attempt
    RETURNING candidate.matches, code.attempts, code.consumed_at
)
SELECT
    COALESCE((SELECT matches FROM updated), false)::boolean AS valid,
    COALESCE(
        (SELECT attempts FROM updated),
        (SELECT attempts FROM candidate),
        0
    )::integer AS attempts,
    COALESCE(
        (SELECT consumed_at IS NOT NULL FROM updated),
        (SELECT consumed_at IS NOT NULL FROM candidate),
        false
    )::boolean AS consumed;

-- name: InsertCaptchaChallenge :exec
INSERT INTO identity_captcha_challenges (
    id, kind, answer_hash, target_x, tolerance, expires_at
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ConsumeCaptchaChallenge :one
UPDATE identity_captcha_challenges AS challenge
SET consumed_at = now()
WHERE challenge.id = $1
  AND challenge.consumed_at IS NULL
  AND challenge.expires_at > now()
  AND (
      (challenge.kind = 'image' AND challenge.answer_hash = $2)
      OR (
          challenge.kind = 'slider'
          AND $3::integer IS NOT NULL
          AND abs(challenge.target_x - $3::integer) <= challenge.tolerance
      )
  )
RETURNING challenge.kind;
