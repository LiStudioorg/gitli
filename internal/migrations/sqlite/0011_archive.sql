-- SPDX-License-Identifier: AGPL-3.0
-- Repository / org archive support (SQLite)

ALTER TABLE repos ADD COLUMN archived_at DATETIME;
ALTER TABLE orgs ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
