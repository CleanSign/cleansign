-- name: UpsertEmailCode :exec
INSERT INTO email_codes (email, code_hash, attempts, sent_at, expires_at, send_count, window_started_at)
VALUES ($1, $2, 0, $3, $4, $5, $6)
ON CONFLICT (email) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
    attempts = 0,
    sent_at = EXCLUDED.sent_at,
    expires_at = EXCLUDED.expires_at,
    send_count = EXCLUDED.send_count,
    window_started_at = EXCLUDED.window_started_at;

-- name: GetEmailCode :one
SELECT * FROM email_codes WHERE email = $1;

-- name: IncrementEmailCodeAttempts :one
UPDATE email_codes SET attempts = attempts + 1 WHERE email = $1 RETURNING attempts;

-- name: DeleteEmailCode :exec
DELETE FROM email_codes WHERE email = $1;

-- name: ExpireEmailCode :exec
UPDATE email_codes SET expires_at = $2 WHERE email = $1;

-- name: DeleteExpiredEmailCodes :execrows
DELETE FROM email_codes WHERE expires_at < $1 AND window_started_at < $2;

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, remember_device, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteSessionsByUser :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < $1;
