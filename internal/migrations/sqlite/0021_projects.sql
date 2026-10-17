-- SPDX-License-Identifier: AGPL-3.0
-- Project boards (SQLite)

CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL,
    repo_id INTEGER,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_columns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name VARCHAR(128) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    column_id INTEGER NOT NULL,
    issue_id INTEGER,
    pull_id INTEGER,
    title VARCHAR(512) NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_projects_repo ON projects (repo_id);
CREATE INDEX idx_project_columns_project ON project_columns (project_id);
CREATE INDEX idx_project_cards_column ON project_cards (column_id);
