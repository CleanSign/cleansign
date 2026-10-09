-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, name, email_verified)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateUserConsent :one
INSERT INTO user_consent (id, user_id, terms, privacy, marketing)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
