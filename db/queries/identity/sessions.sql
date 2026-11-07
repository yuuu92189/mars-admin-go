-- name: CreateSession :one
INSERT INTO identity_sessions (
    id, user_id, client_type, token_hash, refresh_token_hash, device_name,
    ip_address, user_agent, expires_at, refresh_expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
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
UPDATE identity_sessions
SET token_hash = $2,
    refresh_token_hash = $3,
    expires_at = $4,
    refresh_expires_at = $5,
    last_seen_at = now()
WHERE id = $1 AND revoked_at IS NULL AND refresh_expires_at > now()
RETURNING *;

-- name: TouchSession :exec
UPDATE identity_sessions SET last_seen_at = now() WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeSession :execrows
UPDATE identity_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL;

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
INSERT INTO identity_one_time_codes (
    id, purpose, subject, code_hash, expires_at
) VALUES ($1, $2, $3, $4, $5);

-- name: ConsumeOneTimeCode :one
UPDATE identity_one_time_codes AS code
SET consumed_at = now()
WHERE code.id = (
    SELECT candidate.id
    FROM identity_one_time_codes AS candidate
    WHERE candidate.purpose = $1 AND candidate.subject = $2 AND candidate.code_hash = $3
      AND candidate.consumed_at IS NULL AND candidate.expires_at > now() AND candidate.attempts < 5
    ORDER BY candidate.created_at DESC
    LIMIT 1
    FOR UPDATE
)
RETURNING code.*;

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
