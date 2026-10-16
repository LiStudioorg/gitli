-- SPDX-License-Identifier: AGPL-3.0
-- SSH key dual-purpose (auth + sign) and GPG signing keys (SQLite)

ALTER TABLE ssh_keys ADD COLUMN can_authenticate TINYINT(1) NOT NULL DEFAULT 1;
ALTER TABLE ssh_keys ADD COLUMN can_sign TINYINT(1) NOT NULL DEFAULT 0;

CREATE TABLE signing_keys (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    key_type VARCHAR(16) NOT NULL DEFAULT 'gpg',
    title VARCHAR(255) NOT NULL DEFAULT '',
    fingerprint VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    source_key_id BIGINT,
    verified TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_signing_keys_fp ON signing_keys (fingerprint);
CREATE INDEX idx_signing_keys_user ON signing_keys (user_id);
