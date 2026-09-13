-- name: CreateRecoveryCode :exec
INSERT INTO totp_recovery_codes (user_id, code_hash, used) VALUES (?, ?, 0);

-- name: ListRecoveryCodes :many
SELECT * FROM totp_recovery_codes WHERE user_id = ? AND used = 0;

-- name: UseRecoveryCode :exec
UPDATE totp_recovery_codes SET used = 1 WHERE id = ?;

-- name: DeleteRecoveryCodes :exec
DELETE FROM totp_recovery_codes WHERE user_id = ?;

-- name: MarkTOTPUsed :exec
INSERT OR IGNORE INTO totp_used (user_id, "window") VALUES (?, ?);
