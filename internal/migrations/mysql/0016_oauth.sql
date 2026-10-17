-- SPDX-License-Identifier: AGPL-3.0
-- OAuth2 provider (SQLite)

CREATE TABLE oauth_apps (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    owner_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    client_id VARCHAR(64) NOT NULL UNIQUE,
    client_secret_hash VARCHAR(255) NOT NULL,
    redirect_uris TEXT NOT NULL,
    scopes VARCHAR(512) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE oauth_authorizations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    app_id BIGINT NOT NULL,
    scopes VARCHAR(512) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, app_id)
);

CREATE TABLE oauth_codes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    code_hash VARCHAR(255) NOT NULL UNIQUE,
    app_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    redirect_uri VARCHAR(1024) NOT NULL DEFAULT '',
    scopes VARCHAR(512) NOT NULL DEFAULT '',
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE oauth_tokens (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    app_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    scopes VARCHAR(512) NOT NULL DEFAULT '',
    expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth_auth_user ON oauth_authorizations (user_id);
CREATE INDEX idx_oauth_codes_hash ON oauth_codes (code_hash);
CREATE INDEX idx_oauth_tokens_hash ON oauth_tokens (token_hash);
