-- SPDX-License-Identifier: AGPL-3.0
-- Organization IP allowlist (SQLite)

CREATE TABLE ip_allowlist (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    cidr VARCHAR(64) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ip_allowlist_org ON ip_allowlist (org_id);

ALTER TABLE orgs ADD COLUMN ip_allowlist_enabled TINYINT(1) NOT NULL DEFAULT 0;
