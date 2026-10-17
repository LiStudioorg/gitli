-- SPDX-License-Identifier: AGPL-3.0
-- Repository / org archive support (SQLite)

ALTER TABLE repos ADD COLUMN archived_at TIMESTAMPTZ;
ALTER TABLE orgs ADD COLUMN archived BOOLEAN NOT NULL DEFAULT false;
