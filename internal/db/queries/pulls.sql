-- name: CreatePull :exec
INSERT INTO pulls (issue_id, repo_id, head_repo_id, head_branch, base_branch, merged, merge_commit)
VALUES (?, ?, ?, ?, ?, 0, '');

-- name: GetPullByIssue :one
SELECT * FROM pulls WHERE issue_id = ?;

-- name: GetOpenPullByBranches :one
SELECT p.* FROM pulls p
JOIN issues i ON i.id = p.issue_id
WHERE p.repo_id = ?1 AND p.head_branch = ?2 AND p.base_branch = ?3 AND i.closed = 0;

-- name: MergePull :exec
UPDATE pulls SET merged = 1, merge_commit = ? WHERE issue_id = ?;

-- name: ListPullsWithIssue :many
SELECT i.id, i.repo_id, i.number, i.author_id, i.title, i.body, i.is_pull, i.closed,
       i.assignee_id, i.created_at, i.updated_at,
       p.head_branch, p.base_branch, p.merged, p.merge_commit
FROM pulls p
JOIN issues i ON i.id = p.issue_id
WHERE p.repo_id = ?
ORDER BY i.created_at DESC LIMIT 100;
