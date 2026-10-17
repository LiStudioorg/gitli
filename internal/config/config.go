// SPDX-License-Identifier: AGPL-3.0
//
// Package config defines the TOML configuration schema, defaults and loading.
package config

import "time"

// Config is the root configuration decoded from TOML.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Storage  StorageConfig  `toml:"storage"`
	Security SecurityConfig `toml:"security"`
	Modules  ModulesConfig  `toml:"modules"`
	Limits   LimitsConfig   `toml:"limits"`
	Log      LogConfig      `toml:"log"`
	Mail     MailConfig     `toml:"mail"`
	Git      GitConfig      `toml:"git"`
	SSH      SSHConfig      `toml:"ssh"`
	Webhook  WebhookConfig  `toml:"webhook"`
	Backup   BackupConfig   `toml:"backup"`
	OIDC     OIDCConfig     `toml:"oidc"`
	LFS      LFSConfig      `toml:"lfs"`
	CI       CIConfig       `toml:"ci"`
	Search   SearchConfig   `toml:"search"`
	Runner   RunnerConfig   `toml:"runner"`
	Audit    AuditConfig    `toml:"audit"`
	Auth     AuthConfig     `toml:"auth"`
}

// AuthConfig groups external identity providers.
type AuthConfig struct {
	LDAP LDAPConfig `toml:"ldap"`
	SAML SAMLConfig `toml:"saml"`
}

// LDAPConfig configures LDAP authentication.
type LDAPConfig struct {
	Enabled      bool   `toml:"enabled"`
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	BaseDN       string `toml:"base_dn"`
	BindDN       string `toml:"bind_dn"`
	BindPassword string `toml:"bind_password"`
	UserFilter   string `toml:"user_filter"`
	GroupFilter  string `toml:"group_filter"`
	UseTLS       bool   `toml:"use_tls"`
	SkipVerify   bool   `toml:"skip_verify"`
}

// SAMLConfig configures SAML SP-initiated login.
type SAMLConfig struct {
	Enabled        bool   `toml:"enabled"`
	IDPMetadataURL string `toml:"idp_metadata_url"`
	IDPMetadataXML string `toml:"idp_metadata_xml"`
	SPCert         string `toml:"sp_cert"`
	SPKey          string `toml:"sp_key"`
	RootURL        string `toml:"root_url"`
}

// AuditConfig configures audit log retention.
type AuditConfig struct {
	RetentionDays        int `toml:"retention_days"`
	ArchiveRetentionDays int `toml:"archive_retention_days"`
}

// RunnerConfig configures the standalone runner subsystem.
type RunnerConfig struct {
	Enabled                 bool `toml:"enabled"`
	JobTimeoutMinutes       int  `toml:"job_timeout_minutes"`
	LogMaxMB                int  `toml:"log_max_mb"`
	HeartbeatTimeoutSeconds int  `toml:"heartbeat_timeout_seconds"`
	OfflineSweepIntervalSec int  `toml:"offline_sweep_interval_seconds"`
}

// WebhookConfig configures outbound webhook delivery.
type WebhookConfig struct {
	TimeoutSeconds        int  `toml:"timeout_seconds"`
	MaxRetries            int  `toml:"max_retries"`
	RetryIntervalSeconds  int  `toml:"retry_interval_seconds"`
	BlockPrivateAddresses bool `toml:"block_private_addresses"`
}

// BackupConfig configures the backup subsystem.
type BackupConfig struct {
	Encrypt          bool   `toml:"encrypt"`
	EncryptionKeyEnv string `toml:"encryption_key_env"`
	BackupDir        string `toml:"backup_dir"`
}

// OIDCConfig configures OpenID Connect login.
type OIDCConfig struct {
	Enabled      bool     `toml:"enabled"`
	Issuer       string   `toml:"issuer"`
	ClientID     string   `toml:"client_id"`
	ClientSecret string   `toml:"client_secret"`
	RedirectURL  string   `toml:"redirect_url"`
	Scopes       []string `toml:"scopes"`
}

// LFSConfig configures Git Large File Storage.
type LFSConfig struct {
	Enabled     bool   `toml:"enabled"`
	StoragePath string `toml:"storage_path"`
	MaxFileMB   int    `toml:"max_file_mb"`
}

// CIConfig configures the lightweight CI subsystem.
type CIConfig struct {
	Enabled              bool `toml:"enabled"`
	MaxConcurrency       int  `toml:"max_concurrency"`
	RunnerTimeoutSeconds int  `toml:"runner_timeout_seconds"`
	ArtifactMaxMB        int  `toml:"artifact_max_mb"`
}

// SearchConfig configures the search engine backend.
type SearchConfig struct {
	Engine string `toml:"engine"`
}

// SSHConfig configures the SSH git transport behavior. Listen address and the
// enabled switch live under [server] so that listen vs external separation is
// consistent across HTTP and SSH.
type SSHConfig struct {
	Mode                    string `toml:"mode"` // builtin | system
	HostKeyPath             string `toml:"host_key_path"`
	HandshakeTimeoutSeconds int    `toml:"handshake_timeout_seconds"`
	IdleTimeoutSeconds      int    `toml:"idle_timeout_seconds"`
	MaxConnections          int    `toml:"max_connections"`
}

// GitConfig configures the system git binary.
type GitConfig struct {
	Bin            string `toml:"bin"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	CheckVersion   bool   `toml:"check_version"`
	MinVersion     string `toml:"min_version"`
}

// ServerConfig holds listen addresses and externally visible URLs.
//
// Listen addresses and external URLs are deliberately separated: under NAT,
// port mapping or reverse proxying the address the process binds to is not the
// address users must use.
type ServerConfig struct {
	HTTPListen      []string `toml:"http_listen"`
	HTTPExternalURL string   `toml:"http_external_url"`
	SSHEnabled      bool     `toml:"ssh_enabled"`
	SSHListen       []string `toml:"ssh_listen"`
	SSHExternal     string   `toml:"ssh_external"`
	AllowRoot       bool     `toml:"allow_root"`
	EnablePprof     bool     `toml:"pprof"`
	MemoryLimitMB   int      `toml:"memory_limit_mb"`

	// Deprecated single-address fields, accepted for backward compatibility.
	HTTP        string `toml:"http"`
	SSH         string `toml:"ssh"`
	ExternalURL string `toml:"external_url"`
}

// DatabaseConfig selects the SQL driver and connection pool.
type DatabaseConfig struct {
	Driver       string `toml:"driver"`
	DSN          string `toml:"dsn"`
	MaxOpenConns int    `toml:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns"`
}

// StorageConfig holds filesystem roots for persisted objects.
type StorageConfig struct {
	Repos       string `toml:"repos"`
	LFS         string `toml:"lfs"`
	Packages    string `toml:"packages"`
	Avatars     string `toml:"avatars"`
	Attachments string `toml:"attachments"`
}

// SecurityConfig groups password, crypto and login protection settings.
type SecurityConfig struct {
	Password PasswordConfig `toml:"password"`
	Crypto   CryptoConfig   `toml:"crypto"`
	Login    LoginConfig    `toml:"login"`
	Session  SessionConfig  `toml:"session"`
	PAT      PATConfig      `toml:"pat"`
}

// PATConfig configures personal access token policy.
type PATConfig struct {
	AllowClassic       bool `toml:"allow_classic"`
	AllowFineGrained   bool `toml:"allow_fine_grained"`
	DefaultScopesEmpty bool `toml:"default_scopes_empty"`
	MaxExpiryDays      int  `toml:"max_expiry_days"`
}

// PasswordConfig holds Argon2id parameters and password policy.
type PasswordConfig struct {
	Algorithm   string `toml:"algorithm"`
	MemoryMB    uint32 `toml:"memory_mb"`
	Iterations  uint32 `toml:"iterations"`
	Parallelism uint8  `toml:"parallelism"`
	SaltLength  int    `toml:"salt_length"`
	KeyLength   uint32 `toml:"key_length"`
	Pepper      string `toml:"pepper"`
	MinLength   int    `toml:"min_length"`
	HIBP        bool   `toml:"hibp"`
}

// CryptoConfig holds the symmetric key used for secret at-rest encryption.
type CryptoConfig struct {
	KeyHex string `toml:"key_hex"`
}

// LoginConfig configures login rate limiting.
type LoginConfig struct {
	MaxAttempts    int `toml:"max_attempts"`
	LockoutMinutes int `toml:"lockout_minutes"`
	WindowMinutes  int `toml:"window_minutes"`
}

// LockoutDuration returns the lockout window as a duration.
func (l LoginConfig) LockoutDuration() time.Duration {
	return time.Duration(l.LockoutMinutes) * time.Minute
}

// SessionConfig controls session binding and lifetime.
type SessionConfig struct {
	BindIP    bool `toml:"bind_ip"`
	BindAgent bool `toml:"bind_agent"`
	TTLHours  int  `toml:"ttl_hours"`
}

// ModulesConfig toggles feature modules.
type ModulesConfig struct {
	Core          bool           `toml:"core"`
	Collaboration bool           `toml:"collaboration"`
	GitExt        bool           `toml:"git_ext"`
	Registry      bool           `toml:"registry"`
	CI            bool           `toml:"ci"`
	IDE           bool           `toml:"ide"`
	Ops           bool           `toml:"ops"`
	Search        string         `toml:"search"`
	Packages      PackagesModule `toml:"packages"`
}

// PackagesModule lists enabled package registry backends.
type PackagesModule struct {
	Enabled []string `toml:"enabled"`
}

// LimitsConfig holds concurrency and size guardrails.
type LimitsConfig struct {
	MaxGitConcurrency     int           `toml:"max_git_concurrency"`
	MaxWebhookConcurrency int           `toml:"max_webhook_concurrency"`
	MaxCIConcurrency      int           `toml:"max_ci_concurrency"`
	MemLimitMB            int           `toml:"mem_limit_mb"`
	Packages              PackageLimits `toml:"packages"`
}

// PackageLimits bounds package upload sizes.
type PackageLimits struct {
	MaxConcurrent          int `toml:"max_concurrent"`
	GenericMaxFileMB       int `toml:"generic_max_file_mb"`
	NpmMaxTarballMB        int `toml:"npm_max_tarball_mb"`
	NpmMaxMetadataKB       int `toml:"npm_max_metadata_kb"`
	DockerMaxBlobMB        int `toml:"docker_max_blob_mb"`
	DockerMaxManifestKB    int `toml:"docker_max_manifest_kb"`
	DockerUploadTTLMinutes int `toml:"docker_upload_ttl_minutes"`
	GoMaxZipMB             int `toml:"go_max_zip_mb"`
	MavenMaxArtifactMB     int `toml:"maven_max_artifact_mb"`
}

// LogConfig configures structured logging.
type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	// File enables file logging with rotation. Empty keeps logs on stderr.
	File       string `toml:"file"`
	MaxSizeMB  int    `toml:"max_size_mb"`
	MaxBackups int    `toml:"max_backups"`
	MaxAgeDays int    `toml:"max_age_days"`
	Compress   bool   `toml:"compress"`
}

// MailConfig configures outbound SMTP.
type MailConfig struct {
	Enabled  bool              `toml:"enabled"`
	Host     string            `toml:"host"`
	Port     int               `toml:"port"`
	Username string            `toml:"username"`
	Password string            `toml:"password"`
	From     string            `toml:"from"`
	TLS      bool              `toml:"tls"`
	Inbound  InboundMailConfig `toml:"inbound"`
}

// InboundMailConfig configures IMAP reply-by-email ingestion.
type InboundMailConfig struct {
	Enabled     bool   `toml:"enabled"`
	Host        string `toml:"host"`
	Port        int    `toml:"port"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	Mailbox     string `toml:"mailbox"`
	ReplyDomain string `toml:"reply_domain"`
	IntervalSec int    `toml:"interval_seconds"`
}
