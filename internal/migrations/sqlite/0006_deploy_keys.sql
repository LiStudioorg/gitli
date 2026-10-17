-- SPDX-License-Identifier: AGPL-3.0
-- Deploy keys (repository-scoped SSH keys) (SQLite)

CREATE TABLE deploy_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    fingerprint VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    read_only INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE INDEX idx_deploy_keys_repo ON deploy_keys (repo_id);
CREATE INDEX idx_deploy_keys_fp ON deploy_keys (fingerprint);
