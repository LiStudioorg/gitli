-- name: CreateOrg :one
INSERT INTO orgs (name, description, created_at) VALUES (?, ?, ?) RETURNING *;

-- name: GetOrgByName :one
SELECT * FROM orgs WHERE name = ?;

-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, user_id, role, created_at) VALUES (?, ?, ?, ?);

-- name: GetOrgMember :one
SELECT * FROM org_members WHERE org_id = ? AND user_id = ?;

-- name: IsOrgMember :one
SELECT COUNT(*) FROM org_members om
JOIN orgs o ON o.id = om.org_id
WHERE o.name = ?1 AND om.user_id = ?2;

-- name: ListOrgRepos :many
SELECT r.* FROM repos r JOIN orgs o ON r.owner_id = o.id WHERE o.name = ?1;

-- name: CountOrgsOfRepo :one
SELECT COUNT(*) FROM orgs WHERE id = ?;

-- name: GetOrgByID :one
SELECT * FROM orgs WHERE id = ?;

-- name: CountOrgByID :one
SELECT COUNT(*) FROM orgs WHERE id = ?;

-- name: IsOrgMemberByOrgID :one
SELECT COUNT(*) FROM org_members WHERE org_id = ? AND user_id = ?;

-- name: CountOrgMembers :one
SELECT COUNT(*) FROM org_members WHERE org_id = ?;

-- name: ListOrgMembersWithUsers :many
SELECT u.id, u.username, u.email, u.password_hash, u.is_admin, u.created_at, u.updated_at,
       om.role AS member_role
FROM org_members om
JOIN users u ON u.id = om.user_id
WHERE om.org_id = ?
ORDER BY om.created_at;
