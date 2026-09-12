-- name: CountAllUsers :one
SELECT COUNT(*) FROM users;

-- name: CountAllRepos :one
SELECT COUNT(*) FROM repos;

-- name: ListAllUsers :many
SELECT * FROM users ORDER BY created_at ASC;

-- name: ListAllReposWithOwner :many
SELECT r.id, r.owner_id, r.name, r.description, r.visibility, r.created_at, r.updated_at,
       COALESCE(u.username, o.name) AS owner_name
FROM repos r
LEFT JOIN users u ON u.id = r.owner_id
LEFT JOIN orgs o ON o.id = r.owner_id
ORDER BY r.created_at DESC;

-- name: DeleteUserByID :exec
DELETE FROM users WHERE id = ?;
