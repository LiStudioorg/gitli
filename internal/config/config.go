// SPDX-License-Identifier: AGPL-3.0
package config

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
}

// ServerConfig holds HTTP/SSH listener settings.
type ServerConfig struct {
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
}

// PasswordConfig holds Argon2id parameters.
type PasswordConfig struct {
	Algorithm   string `toml:"algorithm"`
	MemoryMB    uint32 `toml:"memory_mb"`
	Iterations  uint32 `toml:"iterations"`
	Parallelism uint8  `toml:"parallelism"`
	SaltLength  int    `toml:"salt_length"`
	KeyLength   uint32 `toml:"key_length"`
	Pepper      string `toml:"pepper"`
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
}

// MailConfig configures outbound SMTP.
type MailConfig struct {
	Enabled  bool   `toml:"enabled"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	From     string `toml:"from"`
	TLS      bool   `toml:"tls"`
}
