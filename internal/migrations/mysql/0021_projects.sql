-- SPDX-License-Identifier: AGPL-3.0
-- Project boards (SQLite)

CREATE TABLE projects (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    repo_id BIGINT,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_columns (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    project_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    position BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_cards (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    column_id BIGINT NOT NULL,
    issue_id BIGINT,
    pull_id BIGINT,
    title VARCHAR(512) NOT NULL DEFAULT '',
    position BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_projects_repo ON projects (repo_id);
CREATE INDEX idx_project_columns_project ON project_columns (project_id);
CREATE INDEX idx_project_cards_column ON project_cards (column_id);
