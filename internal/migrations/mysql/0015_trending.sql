-- SPDX-License-Identifier: AGPL-3.0
-- Trending cache (SQLite)

CREATE TABLE trending_cache (
    repo_id BIGINT PRIMARY KEY,
    score REAL NOT NULL DEFAULT 0,
    computed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trending_score ON trending_cache (score DESC);
