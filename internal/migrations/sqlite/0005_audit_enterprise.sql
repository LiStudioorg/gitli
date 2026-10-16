-- SPDX-License-Identifier: AGPL-3.0
-- Enterprise audit: extended columns, archive table, alert rules (SQLite)

ALTER TABLE audit_logs ADD COLUMN actor_type VARCHAR(16) NOT NULL DEFAULT 'user';
ALTER TABLE audit_logs ADD COLUMN actor_ip VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN actor_user_agent VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN event VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN resource VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN target_name VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN old_value TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN new_value TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN result VARCHAR(16) NOT NULL DEFAULT 'success';
ALTER TABLE audit_logs ADD COLUMN error_message TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN request_id VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE audit_logs ADD COLUMN metadata TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_audit_actor ON audit_logs (actor_id, created_at);
CREATE INDEX idx_audit_event ON audit_logs (event, created_at);
CREATE INDEX idx_audit_target ON audit_logs (target_type, target_id, created_at);
CREATE INDEX idx_audit_result ON audit_logs (result, created_at);

CREATE TABLE audit_logs_archive (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    orig_id INTEGER NOT NULL,
    actor_id INTEGER,
    actor_type VARCHAR(16) NOT NULL DEFAULT 'user',
    actor_ip VARCHAR(64) NOT NULL DEFAULT '',
    actor_user_agent VARCHAR(512) NOT NULL DEFAULT '',
    event VARCHAR(128) NOT NULL DEFAULT '',
    resource VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(64) NOT NULL DEFAULT '',
    target_type VARCHAR(32) NOT NULL DEFAULT '',
    target_id INTEGER,
    target_name VARCHAR(512) NOT NULL DEFAULT '',
    old_value TEXT NOT NULL DEFAULT '',
    new_value TEXT NOT NULL DEFAULT '',
    result VARCHAR(16) NOT NULL DEFAULT 'success',
    error_message TEXT NOT NULL DEFAULT '',
    request_id VARCHAR(64) NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_archive_orig ON audit_logs_archive (orig_id);
CREATE INDEX idx_audit_archive_created ON audit_logs_archive (created_at);

CREATE TABLE audit_alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    event VARCHAR(128) NOT NULL,
    threshold INTEGER NOT NULL DEFAULT 10,
    window_seconds INTEGER NOT NULL DEFAULT 300,
    channel VARCHAR(16) NOT NULL DEFAULT 'email',
    target VARCHAR(1024) NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    last_fired_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
