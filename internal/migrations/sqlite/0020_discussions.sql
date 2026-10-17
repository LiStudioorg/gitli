-- SPDX-License-Identifier: AGPL-3.0
-- Discussions (SQLite)

CREATE TABLE discussion_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_answerable INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    category_id INTEGER,
    title VARCHAR(512) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    author_id INTEGER NOT NULL,
    is_answered INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussion_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    discussion_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_answer INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discussions_repo ON discussions (repo_id, category_id);
CREATE INDEX idx_discussion_comments ON discussion_comments (discussion_id);
CREATE INDEX idx_discussion_categories_repo ON discussion_categories (repo_id);
