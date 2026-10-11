-- SPDX-License-Identifier: AGPL-3.0
-- gitli RBAC schema (SQLite)

ALTER TABLE teams ADD COLUMN parent_team_id INTEGER;
ALTER TABLE teams ADD COLUMN role_id INTEGER;

CREATE TABLE permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (resource, action)
);

CREATE TABLE roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    scope VARCHAR(32) NOT NULL DEFAULT 'org',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (org_id, name)
);

CREATE TABLE role_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (role_id, permission_id)
);

CREATE TABLE org_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role_id INTEGER,
    membership VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (org_id, user_id)
);

CREATE TABLE team_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    repo_id INTEGER NOT NULL,
    role_id INTEGER,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (team_id, repo_id)
);

CREATE TABLE repo_collaborators (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role_id INTEGER,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (repo_id, user_id)
);

CREATE TABLE branch_protection_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    pattern VARCHAR(255) NOT NULL,
    require_reviews INTEGER NOT NULL DEFAULT 0,
    require_signed INTEGER NOT NULL DEFAULT 0,
    require_ci_jobs VARCHAR(512) NOT NULL DEFAULT '',
    push_roles TEXT NOT NULL DEFAULT '',
    merge_roles TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE path_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    path_pattern VARCHAR(1024) NOT NULL,
    role_id INTEGER,
    permission VARCHAR(16) NOT NULL DEFAULT 'read',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permission_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id INTEGER,
    target_type VARCHAR(64) NOT NULL DEFAULT '',
    target_id INTEGER,
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
