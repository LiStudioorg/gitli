-- name: CreatePAT :one
INSERT INTO pats (user_id, name, token_hash, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetPATByHash :one
SELECT * FROM pats WHERE token_hash = ?;

-- name: ListPATsByUser :many
SELECT id, user_id, name, token_hash, created_at, last_used_at FROM pats WHERE user_id = ?;

-- name: DeletePAT :exec
DELETE FROM pats WHERE id = ? AND user_id = ?;

-- name: TouchPAT :exec
UPDATE pats SET last_used_at = ? WHERE id = ?;
