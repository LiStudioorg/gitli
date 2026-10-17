// SPDX-License-Identifier: AGPL-3.0
package main

import (
	"fmt"
	"strings"

	"github.com/LiStudioorg/gitli/internal/config"
)

// weakSecrets are values that have appeared in public docs, example units or
// default templates. Using them provides no protection at all, so we refuse to
// start instead of silently encrypting TOTP secrets with a known key.
var weakSecrets = map[string]string{
	"change-me": "示例占位值",
	"changeme":  "示例占位值",
	"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef": "公开文档中的示例密钥",
}

// checkWeakSecrets rejects well-known placeholder secrets.
func checkWeakSecrets(cfg *config.Config) error {
	if msg, err := weakSecretProblem("GITLI_CRYPTO_KEY", cfg.Security.Crypto.KeyHex); err != nil {
		return err
	} else if msg != "" {
		return fmt.Errorf("密钥不安全：%s 使用了%s。请用 `gitli admin generate-secret` 生成新密钥后重启", "GITLI_CRYPTO_KEY", msg)
	}
	if msg, err := weakSecretProblem("GITLI_PEPPER", cfg.Security.Password.Pepper); err != nil {
		return err
	} else if msg != "" {
		return fmt.Errorf("密钥不安全：%s 使用了%s。请用 `gitli admin generate-secret` 生成新密钥后重启", "GITLI_PEPPER", msg)
	}
	return nil
}

// weakSecretProblem reports a human-readable reason when value is a known
// placeholder, contains no entropy (all same character) or is too short.
func weakSecretProblem(name, value string) (string, error) {
	if value == "" {
		return "", nil // empty means "not configured"; handled elsewhere
	}
	if reason, ok := weakSecrets[strings.ToLower(strings.TrimSpace(value))]; ok {
		return reason, nil
	}
	if isLowEntropy(value) {
		return "低熵值（全部字符相同或为连续序列）", nil
	}
	if len(value) < 16 {
		return "长度不足 16 个字符", nil
	}
	return "", nil
}

func isLowEntropy(v string) bool {
	if v == "" {
		return false
	}
	first := v[0]
	same := true
	for i := 1; i < len(v); i++ {
		if v[i] != first {
			same = false
			break
		}
	}
	if same {
		return true
	}
	// Detect simple ascending runs such as "0123456789abcdef".
	ascending := true
	for i := 1; i < len(v); i++ {
		if v[i] != v[i-1]+1 {
			ascending = false
			break
		}
	}
	return ascending
}
