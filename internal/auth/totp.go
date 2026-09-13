package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"gitli/internal/db"
)

// TOTP 相关。

// GenerateTOTPSecret 生成新 TOTP 密钥（base32 无填充）。
func GenerateTOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "gitli",
		AccountName: "setup",
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1, // 兼容所有验证器 App
		SecretSize:  20,
	})
	if err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return key.Secret(), nil
}

// TOTPProvisioningURI 生成 otpauth:// 链接（供验证器扫码/手输）。
func TOTPProvisioningURI(secret, account string) string {
	return fmt.Sprintf("otpauth://totp/gitli:%s?secret=%s&issuer=gitli&period=30&digits=6&algorithm=SHA1",
		account, secret)
}

// VerifyTOTP 校验 6 位验证码（±1 窗口容错），成功则记录防重放窗口。
func VerifyTOTP(ctx context.Context, q db.Querier, userID int64, secret, code string) bool {
	if secret == "" || len(code) != 6 {
		return false
	}
	now := time.Now().UTC()
	ok, err := totp.ValidateCustom(code, secret, now, totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !ok {
		return false
	}
	// 防重放：记录当前窗口（允许过去 1 个窗口）
	for _, w := range []int64{now.Unix() / 30, now.Unix()/30 - 1} {
		_ = q.MarkTOTPUsed(ctx, db.MarkTOTPUsedParams{UserID: userID, Window: w})
	}
	return true
}

// GenerateRecoveryCodes 生成 10 个备用码，返回明文列表（仅此一次可见），哈希入库。
func GenerateRecoveryCodes(ctx context.Context, q db.Querier, userID int64) ([]string, error) {
	_ = q.DeleteRecoveryCodes(ctx, userID)
	codes := make([]string, 10)
	for i := range codes {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("rand: %w", err)
		}
		c := hex.EncodeToString(b)[:8]
		codes[i] = c
		if err := q.CreateRecoveryCode(ctx, db.CreateRecoveryCodeParams{
			UserID: userID, CodeHash: hashToken(c),
		}); err != nil {
			return nil, fmt.Errorf("save recovery code: %w", err)
		}
	}
	return codes, nil
}

// UseRecoveryCode 校验并消费一个备用码。
func UseRecoveryCode(ctx context.Context, q db.Querier, userID int64, code string) bool {
	remaining, err := q.ListRecoveryCodes(ctx, userID)
	if err != nil {
		return false
	}
	for _, rc := range remaining {
		if subtle.ConstantTimeCompare([]byte(rc.CodeHash), []byte(hashToken(strings.TrimSpace(code)))) == 1 {
			_ = q.UseRecoveryCode(ctx, rc.ID)
			return true
		}
	}
	return false
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// normalizeSecret 兼容粘贴带空格/小写的 secret。
func normalizeSecret(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, " ", ""))
}
