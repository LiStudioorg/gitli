// SPDX-License-Identifier: AGPL-3.0
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Load reads the TOML file at path, applying defaults and environment overrides.
// A missing file is not an error: defaults plus environment are returned.
func Load(path string) (*Config, error) {
	cfg := Defaults()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, cfg); err != nil {
				return nil, fmt.Errorf("decode config %s: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat config %s: %w", path, err)
		}
	}
	applyEnvOverrides(cfg)
	return cfg, nil
}

// WriteDefault writes a default config file to path if it does not exist.
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(defaultTemplate), 0o600)
}

func applyEnvOverrides(cfg *Config) {
	setStr(&cfg.Server.HTTP, "GITLI_SERVER_HTTP")
	setStr(&cfg.Server.SSH, "GITLI_SERVER_SSH")
	setStr(&cfg.Server.ExternalURL, "GITLI_SERVER_EXTERNAL_URL")

	setStr(&cfg.Database.Driver, "GITLI_DATABASE_DRIVER")
	setStr(&cfg.Database.DSN, "GITLI_DATABASE_DSN")
	setInt(&cfg.Database.MaxOpenConns, "GITLI_DATABASE_MAX_OPEN_CONNS")
	setInt(&cfg.Database.MaxIdleConns, "GITLI_DATABASE_MAX_IDLE_CONNS")

	setStr(&cfg.Storage.Repos, "GITLI_STORAGE_REPOS")
	setStr(&cfg.Storage.LFS, "GITLI_STORAGE_LFS")
	setStr(&cfg.Storage.Packages, "GITLI_STORAGE_PACKAGES")
	setStr(&cfg.Storage.Avatars, "GITLI_STORAGE_AVATARS")
	setStr(&cfg.Storage.Attachments, "GITLI_STORAGE_ATTACHMENTS")

	setStr(&cfg.Security.Password.Pepper, "GITLI_SECURITY_PASSWORD_PEPPER")
	setStr(&cfg.Security.Password.Pepper, "GITLI_PEPPER")
	setStr(&cfg.Security.Crypto.KeyHex, "GITLI_SECURITY_CRYPTO_KEY")
	setStr(&cfg.Security.Crypto.KeyHex, "GITLI_CRYPTO_KEY")
	setInt(&cfg.Security.Login.MaxAttempts, "GITLI_SECURITY_LOGIN_MAX_ATTEMPTS")
	setInt(&cfg.Security.Login.LockoutMinutes, "GITLI_SECURITY_LOGIN_LOCKOUT_MINUTES")

	setStr(&cfg.Modules.Search, "GITLI_MODULES_SEARCH")

	setStr(&cfg.Log.Level, "GITLI_LOG_LEVEL")
	setStr(&cfg.Log.Format, "GITLI_LOG_FORMAT")

	setBool(&cfg.Mail.Enabled, "GITLI_MAIL_ENABLED")
	setStr(&cfg.Mail.Host, "GITLI_MAIL_HOST")
	setInt(&cfg.Mail.Port, "GITLI_MAIL_PORT")
	setStr(&cfg.Mail.Username, "GITLI_MAIL_USERNAME")
	setStr(&cfg.Mail.Password, "GITLI_MAIL_PASSWORD")
	setStr(&cfg.Mail.From, "GITLI_MAIL_FROM")
	setBool(&cfg.Mail.TLS, "GITLI_MAIL_TLS")
}

func setStr(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		*dst = v
	}
}

func setInt(dst *int, key string) {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}

func setBool(dst *bool, key string) {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			*dst = b
		}
	}
}
