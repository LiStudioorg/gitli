-- SPDX-License-Identifier: AGPL-3.0
-- Discussions (SQLite)

CREATE TABLE discussion_categories (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_answerable BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussions (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    category_id BIGINT,
    title VARCHAR(512) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    author_id BIGINT NOT NULL,
    is_answered BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discussion_comments (
    id BIGSERIAL PRIMARY KEY,
    discussion_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_answer BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discussions_repo ON discussions (repo_id, category_id);
CREATE INDEX idx_discussion_comments ON discussion_comments (discussion_id);
CREATE INDEX idx_discussion_categories_repo ON discussion_categories (repo_id);
