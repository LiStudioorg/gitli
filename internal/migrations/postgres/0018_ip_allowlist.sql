-- SPDX-License-Identifier: AGPL-3.0
-- Organization IP allowlist (SQLite)

CREATE TABLE ip_allowlist (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    cidr VARCHAR(64) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ip_allowlist_org ON ip_allowlist (org_id);

ALTER TABLE orgs ADD COLUMN ip_allowlist_enabled BOOLEAN NOT NULL DEFAULT false;
