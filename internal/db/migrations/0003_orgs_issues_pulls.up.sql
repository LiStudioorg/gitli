-- 组织
CREATE TABLE orgs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);

CREATE TABLE org_members (
    org_id     INTEGER NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner','member')),
    created_at INTEGER NOT NULL,
    PRIMARY KEY (org_id, user_id)
);

-- SSH 公钥
CREATE TABLE ssh_keys (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    fingerprint TEXT NOT NULL UNIQUE,
    public_key  TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    UNIQUE (user_id, name)
);

-- Issue / PR 公共表（is_pull 区分）
CREATE TABLE issues (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id    INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    number     INTEGER NOT NULL,
    author_id  INTEGER NOT NULL REFERENCES users(id),
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    is_pull    INTEGER NOT NULL DEFAULT 0,
    closed     INTEGER NOT NULL DEFAULT 0,
    assignee_id INTEGER REFERENCES users(id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (repo_id, number)
);
CREATE INDEX idx_issues_repo ON issues(repo_id, is_pull, closed);

CREATE TABLE issue_comments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id  INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id),
    body      TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_comments_issue ON issue_comments(issue_id);

CREATE TABLE labels (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    name    TEXT NOT NULL,
    color   TEXT NOT NULL DEFAULT '#58a6ff',
    UNIQUE (repo_id, name)
);

CREATE TABLE issue_labels (
    issue_id INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id INTEGER NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

-- PR 扩展信息（issue_id 复用 issues 行）
CREATE TABLE pulls (
    issue_id     INTEGER PRIMARY KEY REFERENCES issues(id) ON DELETE CASCADE,
    repo_id      INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    head_repo_id INTEGER NOT NULL REFERENCES repos(id),
    head_branch  TEXT NOT NULL,
    base_branch  TEXT NOT NULL,
    merged       INTEGER NOT NULL DEFAULT 0,
    merge_commit TEXT
);
