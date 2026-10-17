-- SPDX-License-Identifier: AGPL-3.0
-- Detected SPDX license id (SQLite)

ALTER TABLE repos ADD COLUMN license VARCHAR(64) NOT NULL DEFAULT '';
