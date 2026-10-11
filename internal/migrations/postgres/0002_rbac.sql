-- SPDX-License-Identifier: AGPL-3.0
-- gitli RBAC schema (SQLite)

ALTER TABLE teams ADD COLUMN parent_team_id BIGINT;
ALTER TABLE teams ADD COLUMN role_id BIGINT;

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    resource VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (resource, action)
);

CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_builtin BOOLEAN NOT NULL DEFAULT false,
    scope VARCHAR(32) NOT NULL DEFAULT 'org',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (org_id, name)
);

CREATE TABLE role_permissions (
    id BIGSERIAL PRIMARY KEY,
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (role_id, permission_id)
);

CREATE TABLE org_members (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_id BIGINT,
    membership VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (org_id, user_id)
);

CREATE TABLE team_repos (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL,
    repo_id BIGINT NOT NULL,
    role_id BIGINT,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (team_id, repo_id)
);

CREATE TABLE repo_collaborators (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_id BIGINT,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (repo_id, user_id)
);

CREATE TABLE branch_protection_rules (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    pattern VARCHAR(255) NOT NULL,
    require_reviews BOOLEAN NOT NULL DEFAULT false,
    require_signed BOOLEAN NOT NULL DEFAULT false,
    require_ci_jobs VARCHAR(512) NOT NULL DEFAULT '',
    push_roles TEXT NOT NULL DEFAULT '',
    merge_roles TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE path_permissions (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    path_pattern VARCHAR(1024) NOT NULL,
    role_id BIGINT,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permission_audit (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT,
    target_type VARCHAR(64) NOT NULL DEFAULT '',
    target_id BIGINT,
    action VARCHAR(64) NOT NULL,
    old_value TEXT NOT NULL DEFAULT '',
    new_value TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_org_members_org ON org_members (org_id);
CREATE INDEX idx_org_members_user ON org_members (user_id);
CREATE INDEX idx_role_permissions_role ON role_permissions (role_id);
CREATE INDEX idx_team_repos_team ON team_repos (team_id);
CREATE INDEX idx_repo_collaborators_user ON repo_collaborators (user_id);
CREATE INDEX idx_permission_audit_target ON permission_audit (target_type, target_id);
