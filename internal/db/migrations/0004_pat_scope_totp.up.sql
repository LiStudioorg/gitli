-- PAT 增加仓库级作用域：scope = 'all' 或 'repo:<repo_id>:<read|write>'
ALTER TABLE pats ADD COLUMN scope TEXT NOT NULL DEFAULT 'all';
ALTER TABLE pats ADD COLUMN expires_at INTEGER;

-- 用户 TOTP 两步验证
ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;

-- TOTP 恢复码（哈希存储）
CREATE TABLE totp_recovery_codes (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_recovery_user ON totp_recovery_codes(user_id);

-- 已用过的 TOTP 窗口防重放
CREATE TABLE totp_used (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    window   INTEGER NOT NULL,
    PRIMARY KEY (user_id, window)
);
