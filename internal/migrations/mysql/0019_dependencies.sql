-- SPDX-License-Identifier: AGPL-3.0
-- Repository dependency graph (SQLite)

CREATE TABLE repo_dependencies (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    ecosystem VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(128) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_repo_deps_repo ON repo_dependencies (repo_id, ecosystem);
