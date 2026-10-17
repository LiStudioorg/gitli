-- SPDX-License-Identifier: AGPL-3.0
-- Repository soft delete (SQLite)

ALTER TABLE repos ADD COLUMN deleted_at DATETIME;
CREATE INDEX idx_repos_deleted ON repos (deleted_at);
