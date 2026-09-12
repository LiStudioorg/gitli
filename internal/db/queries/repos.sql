-- name: CreateRepo :one
INSERT INTO repos (owner_id, name, description, visibility, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRepoByID :one
SELECT * FROM repos WHERE id = ?;

-- name: GetRepoByOwnerAndName :one
SELECT r.* FROM repos r
JOIN users u ON u.id = r.owner_id
WHERE u.username = ? AND r.name = ?;

-- name: ListReposByOwner :many
SELECT r.* FROM repos r
JOIN users u ON u.id = r.owner_id
WHERE u.username = ?
ORDER BY r.created_at DESC;

-- name: ListPublicRepos :many
SELECT * FROM repos WHERE visibility = 'public'
ORDER BY created_at DESC LIMIT 100;

-- name: SearchRepos :many
SELECT * FROM repos
WHERE (name LIKE '%' || ? || '%' OR description LIKE '%' || ? || '%')
ORDER BY created_at DESC LIMIT 50;

-- name: UpdateRepo :exec
UPDATE repos SET description = ?, visibility = ?, updated_at = ? WHERE id = ?;

-- name: DeleteRepo :exec
DELETE FROM repos WHERE id = ?;

-- name: CountReposByOwner :one
SELECT COUNT(*) FROM repos WHERE owner_id = ?;

-- name: GetRepoByOrgAndName :one
SELECT r.* FROM repos r
JOIN orgs o ON o.id = r.owner_id
WHERE o.name = ? AND r.name = ?;
