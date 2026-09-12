-- name: AddCollaborator :exec
INSERT INTO collaborators (repo_id, user_id, created_at) VALUES (?, ?, ?);

-- name: RemoveCollaborator :exec
DELETE FROM collaborators WHERE repo_id = ? AND user_id = ?;

-- name: IsCollaborator :one
SELECT COUNT(*) FROM collaborators WHERE repo_id = ? AND user_id = ?;

-- name: ListCollaborators :many
SELECT u.* FROM collaborators c
JOIN users u ON u.id = c.user_id
WHERE c.repo_id = ?;

-- name: ListVisibleReposForUser :many
-- 用户可见的仓库：自己的 + public + 协作的（org 可见性 M5 补组织成员查询）
SELECT r.* FROM repos r
LEFT JOIN collaborators c ON c.repo_id = r.id AND c.user_id = ?
WHERE r.visibility = 'public'
   OR r.owner_id = ?
   OR c.user_id IS NOT NULL
ORDER BY r.created_at DESC LIMIT 200;
