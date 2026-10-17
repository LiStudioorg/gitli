-- SPDX-License-Identifier: AGPL-3.0
-- Notification email digest preferences (SQLite)

CREATE TABLE user_notification_prefs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    frequency VARCHAR(16) NOT NULL DEFAULT 'realtime',
    last_sent_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
