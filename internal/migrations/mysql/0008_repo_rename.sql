-- SPDX-License-Identifier: AGPL-3.0
-- Repository rename redirects (SQLite)

CREATE TABLE repo_redirects (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    old_owner VARCHAR(255) NOT NULL,
    old_name VARCHAR(255) NOT NULL,
    repo_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (old_owner, old_name)
);

CREATE INDEX idx_repo_redirects_repo ON repo_redirects (repo_id);
