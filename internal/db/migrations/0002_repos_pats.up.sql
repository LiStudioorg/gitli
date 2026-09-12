-- 仓库表
CREATE TABLE repos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    visibility  TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','org','private')),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    UNIQUE(owner_id, name)
);
CREATE INDEX idx_repos_owner_id ON repos(owner_id);

-- 协作者表
CREATE TABLE collaborators (
    repo_id     INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (repo_id, user_id)
);

-- Personal Access Token 表（只存 SHA-256 hex）
CREATE TABLE pats (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    last_used_at INTEGER
);
CREATE INDEX idx_pats_user_id ON pats(user_id);
