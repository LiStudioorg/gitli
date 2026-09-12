-- name: NextIssueNumber :one
SELECT COALESCE(MAX(number), 0) + 1 AS n FROM issues WHERE repo_id = ?;

-- name: CreateIssue :one
INSERT INTO issues (repo_id, number, author_id, title, body, is_pull, closed, assignee_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?) RETURNING *;

-- name: GetIssue :one
SELECT * FROM issues WHERE repo_id = ? AND number = ?;

-- name: GetIssueByID :one
SELECT * FROM issues WHERE id = ?;

-- name: ListIssues :many
SELECT * FROM issues WHERE repo_id = ?1 AND is_pull = ?2
ORDER BY created_at DESC LIMIT 100;

-- name: ListIssuesByState :many
SELECT * FROM issues WHERE repo_id = ?1 AND is_pull = ?2 AND closed = ?3
ORDER BY created_at DESC LIMIT 100;

-- name: UpdateIssueState :exec
UPDATE issues SET closed = ?, updated_at = ? WHERE id = ?;

-- name: UpdateIssueAssignee :exec
UPDATE issues SET assignee_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateIssueTitleBody :exec
UPDATE issues SET title = ?, body = ?, updated_at = ? WHERE id = ?;

-- name: CreateComment :one
INSERT INTO issue_comments (issue_id, author_id, body, created_at)
VALUES (?, ?, ?, ?) RETURNING *;

-- name: ListComments :many
SELECT * FROM issue_comments WHERE issue_id = ? ORDER BY created_at;

-- name: CountComments :one
SELECT COUNT(*) FROM issue_comments WHERE issue_id = ?;

-- name: AddLabel :exec
INSERT INTO issue_labels (issue_id, label_id) VALUES (?, ?);

-- name: RemoveLabel :exec
DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?;

-- name: ListIssueLabels :many
SELECT l.* FROM issue_labels il JOIN labels l ON l.id = il.label_id WHERE il.issue_id = ?;

-- name: CreateLabel :one
INSERT INTO labels (repo_id, name, color) VALUES (?, ?, ?) RETURNING *;

-- name: ListLabels :many
SELECT * FROM labels WHERE repo_id = ?;

-- name: GetLabelByName :one
SELECT * FROM labels WHERE repo_id = ? AND name = ?;
