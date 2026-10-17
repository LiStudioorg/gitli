-- SPDX-License-Identifier: AGPL-3.0
-- Notification email digest preferences (SQLite)

CREATE TABLE user_notification_prefs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    frequency VARCHAR(16) NOT NULL DEFAULT 'realtime',
    last_sent_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
