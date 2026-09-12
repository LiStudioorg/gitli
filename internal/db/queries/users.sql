-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, is_admin, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: SearchUsers :many
SELECT * FROM users
WHERE username LIKE '%' || ? || '%'
ORDER BY username
LIMIT 50;

-- name: UpdateUserEmail :exec
UPDATE users SET email = ?, updated_at = ? WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
