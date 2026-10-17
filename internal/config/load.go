// SPDX-License-Identifier: AGPL-3.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	resolvePaths(cfg, path)
	return cfg, nil
}

// resolvePaths makes filesystem paths absolute, resolved relative to the config
// file's directory. This keeps git hooks (which run with an arbitrary CWD)
// working regardless of where git invokes them.
func resolvePaths(cfg *Config, configPath string) {
	base := "."
	if configPath != "" {
		if abs, err := filepath.Abs(configPath); err == nil {
			base = filepath.Dir(abs)
		}
	}
	abs := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(base, p)
	}
	if cfg.Database.Driver == "sqlite" || cfg.Database.Driver == "" {
		if cfg.Database.DSN != "" && !strings.Contains(cfg.Database.DSN, "://") && !strings.HasPrefix(cfg.Database.DSN, ":") {
			// Strip query params before path resolution and re-attach.
			dsnPath := cfg.Database.DSN
			suffix := ""
			if i := strings.IndexByte(dsnPath, '?'); i >= 0 {
				dsnPath, suffix = dsnPath[:i], dsnPath[i:]
			}
			cfg.Database.DSN = abs(dsnPath) + suffix
		}
	}
	cfg.Storage.Repos = abs(cfg.Storage.Repos)
	cfg.Storage.LFS = abs(cfg.Storage.LFS)
	cfg.Storage.Packages = abs(cfg.Storage.Packages)
	cfg.Storage.Avatars = abs(cfg.Storage.Avatars)
	cfg.Storage.Attachments = abs(cfg.Storage.Attachments)
	cfg.SSH.HostKeyPath = abs(cfg.SSH.HostKeyPath)
	cfg.Backup.BackupDir = abs(cfg.Backup.BackupDir)
}

// WriteDefault writes a default config file to path if it does not exist.
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(defaultTemplate), 0o600)
}

func applyEnvOverrides(cfg *Config) {
	setStrSlice(&cfg.Server.HTTPListen, "GITLI_SERVER_HTTP_LISTEN")
	setStr(&cfg.Server.HTTPExternalURL, "GITLI_SERVER_HTTP_EXTERNAL_URL")
	setBool(&cfg.Server.SSHEnabled, "GITLI_SERVER_SSH_ENABLED")
	setStrSlice(&cfg.Server.SSHListen, "GITLI_SERVER_SSH_LISTEN")
	setStr(&cfg.Server.SSHExternal, "GITLI_SERVER_SSH_EXTERNAL")
	setBool(&cfg.Server.AllowRoot, "GITLI_SERVER_ALLOW_ROOT")

	// Legacy single-address env vars.
	setStr(&cfg.Server.HTTP, "GITLI_SERVER_HTTP")
	setStr(&cfg.Server.SSH, "GITLI_SERVER_SSH")
	setStr(&cfg.Server.ExternalURL, "GITLI_SERVER_EXTERNAL_URL")

	migrateLegacyServer(cfg)

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
	setInt(&cfg.Security.Password.MinLength, "GITLI_SECURITY_PASSWORD_MIN_LENGTH")
	setBool(&cfg.Security.Password.HIBP, "GITLI_SECURITY_PASSWORD_HIBP")
	setInt(&cfg.Security.Login.MaxAttempts, "GITLI_SECURITY_LOGIN_MAX_ATTEMPTS")
	setInt(&cfg.Security.Login.LockoutMinutes, "GITLI_SECURITY_LOGIN_LOCKOUT_MINUTES")
	setBool(&cfg.Security.Session.BindIP, "GITLI_SECURITY_SESSION_BIND_IP")
	setBool(&cfg.Security.Session.BindAgent, "GITLI_SECURITY_SESSION_BIND_AGENT")

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

	setStr(&cfg.Git.Bin, "GITLI_GIT_BIN")
	setInt(&cfg.Git.TimeoutSeconds, "GITLI_GIT_TIMEOUT_SECONDS")
	setBool(&cfg.Git.CheckVersion, "GITLI_GIT_CHECK_VERSION")
	setStr(&cfg.Git.MinVersion, "GITLI_GIT_MIN_VERSION")

	setStr(&cfg.SSH.Mode, "GITLI_SSH_MODE")
	setStr(&cfg.SSH.HostKeyPath, "GITLI_SSH_HOST_KEY_PATH")
	setInt(&cfg.SSH.HandshakeTimeoutSeconds, "GITLI_SSH_HANDSHAKE_TIMEOUT_SECONDS")
	setInt(&cfg.SSH.IdleTimeoutSeconds, "GITLI_SSH_IDLE_TIMEOUT_SECONDS")
	setInt(&cfg.SSH.MaxConnections, "GITLI_SSH_MAX_CONNECTIONS")

	setInt(&cfg.Webhook.TimeoutSeconds, "GITLI_WEBHOOK_TIMEOUT_SECONDS")
	setInt(&cfg.Webhook.MaxRetries, "GITLI_WEBHOOK_MAX_RETRIES")
	setBool(&cfg.Webhook.BlockPrivateAddresses, "GITLI_WEBHOOK_BLOCK_PRIVATE_ADDRESSES")

	setBool(&cfg.Backup.Encrypt, "GITLI_BACKUP_ENCRYPT")
	setStr(&cfg.Backup.BackupDir, "GITLI_BACKUP_DIR")

	setBool(&cfg.OIDC.Enabled, "GITLI_OIDC_ENABLED")
	setStr(&cfg.OIDC.Issuer, "GITLI_OIDC_ISSUER")
	setStr(&cfg.OIDC.ClientID, "GITLI_OIDC_CLIENT_ID")
	setStr(&cfg.OIDC.ClientSecret, "GITLI_OIDC_CLIENT_SECRET")
	setStr(&cfg.OIDC.RedirectURL, "GITLI_OIDC_REDIRECT_URL")

	setBool(&cfg.LFS.Enabled, "GITLI_LFS_ENABLED")
	setStr(&cfg.LFS.StoragePath, "GITLI_LFS_STORAGE_PATH")
	setInt(&cfg.LFS.MaxFileMB, "GITLI_LFS_MAX_FILE_MB")

	setBool(&cfg.CI.Enabled, "GITLI_CI_ENABLED")
	setInt(&cfg.CI.MaxConcurrency, "GITLI_CI_MAX_CONCURRENCY")
	setInt(&cfg.CI.RunnerTimeoutSeconds, "GITLI_CI_RUNNER_TIMEOUT_SECONDS")
	setInt(&cfg.CI.ArtifactMaxMB, "GITLI_CI_ARTIFACT_MAX_MB")

	setStr(&cfg.Search.Engine, "GITLI_SEARCH_ENGINE")
	setStr(&cfg.Modules.Search, "GITLI_MODULES_SEARCH")

	setBool(&cfg.Auth.LDAP.Enabled, "GITLI_AUTH_LDAP_ENABLED")
	setStr(&cfg.Auth.LDAP.Host, "GITLI_AUTH_LDAP_HOST")
	setInt(&cfg.Auth.LDAP.Port, "GITLI_AUTH_LDAP_PORT")
	setStr(&cfg.Auth.LDAP.BaseDN, "GITLI_AUTH_LDAP_BASE_DN")
	setStr(&cfg.Auth.LDAP.BindDN, "GITLI_AUTH_LDAP_BIND_DN")
	setStr(&cfg.Auth.LDAP.BindPassword, "GITLI_AUTH_LDAP_BIND_PASSWORD")
	setStr(&cfg.Auth.LDAP.UserFilter, "GITLI_AUTH_LDAP_USER_FILTER")

	setBool(&cfg.Auth.SAML.Enabled, "GITLI_AUTH_SAML_ENABLED")
	setStr(&cfg.Auth.SAML.IDPMetadataURL, "GITLI_AUTH_SAML_IDP_METADATA_URL")
	setStr(&cfg.Auth.SAML.RootURL, "GITLI_AUTH_SAML_ROOT_URL")

	setBool(&cfg.Runner.Enabled, "GITLI_RUNNER_ENABLED")
	setInt(&cfg.Runner.JobTimeoutMinutes, "GITLI_RUNNER_JOB_TIMEOUT_MINUTES")
	setInt(&cfg.Runner.LogMaxMB, "GITLI_RUNNER_LOG_MAX_MB")
	setInt(&cfg.Runner.HeartbeatTimeoutSeconds, "GITLI_RUNNER_HEARTBEAT_TIMEOUT_SECONDS")
	setInt(&cfg.Runner.OfflineSweepIntervalSec, "GITLI_RUNNER_OFFLINE_SWEEP_INTERVAL_SECONDS")
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
