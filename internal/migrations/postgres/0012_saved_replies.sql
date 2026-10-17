-- SPDX-License-Identifier: AGPL-3.0
-- Saved replies (SQLite)

CREATE TABLE saved_replies (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_saved_replies_user ON saved_replies (user_id);
