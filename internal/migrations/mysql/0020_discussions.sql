-- SPDX-License-Identifier: AGPL-3.0
-- Discussions (SQLite)

CREATE TABLE discussion_categories (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_answerable TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    category_id BIGINT,
    title VARCHAR(512) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    author_id BIGINT NOT NULL,
    is_answered TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussion_comments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    discussion_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_answer TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discussions_repo ON discussions (repo_id, category_id);
CREATE INDEX idx_discussion_comments ON discussion_comments (discussion_id);
CREATE INDEX idx_discussion_categories_repo ON discussion_categories (repo_id);
