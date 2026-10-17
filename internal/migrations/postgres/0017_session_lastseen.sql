-- SPDX-License-Identifier: AGPL-3.0
-- Session last-seen tracking (PostgreSQL)

ALTER TABLE sessions ADD COLUMN last_seen_at TIMESTAMPTZ;
