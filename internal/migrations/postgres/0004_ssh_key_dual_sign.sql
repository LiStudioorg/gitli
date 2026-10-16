-- SPDX-License-Identifier: AGPL-3.0
-- SSH key dual-purpose (auth + sign) and GPG signing keys (SQLite)

ALTER TABLE ssh_keys ADD COLUMN can_authenticate BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE ssh_keys ADD COLUMN can_sign BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE signing_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    key_type VARCHAR(16) NOT NULL DEFAULT 'gpg',
    title VARCHAR(255) NOT NULL DEFAULT '',
    fingerprint VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    source_key_id BIGINT,
    verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_signing_keys_fp ON signing_keys (fingerprint);
CREATE INDEX idx_signing_keys_user ON signing_keys (user_id);
