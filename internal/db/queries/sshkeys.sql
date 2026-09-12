-- name: CreateSSHKey :one
INSERT INTO ssh_keys (user_id, name, fingerprint, public_key, created_at)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: ListSSHKeysByUser :many
SELECT * FROM ssh_keys WHERE user_id = ?;

-- name: GetSSHKeyByFingerprint :one
SELECT * FROM ssh_keys WHERE fingerprint = ?;

-- name: DeleteSSHKey :exec
DELETE FROM ssh_keys WHERE id = ? AND user_id = ?;
