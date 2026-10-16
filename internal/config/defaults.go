// SPDX-License-Identifier: AGPL-3.0
package config

// Defaults returns a configuration populated with safe defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPListen:      []string{"0.0.0.0:8080", "[::]:8080"},
			HTTPExternalURL: "",
			SSHEnabled:      true,
			SSHListen:       []string{"0.0.0.0:2222", "[::]:2222"},
			SSHExternal:     "",
		},
		Database: DatabaseConfig{
			Driver:       "sqlite",
			DSN:          "./data/gitli.db",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Storage: StorageConfig{
			Repos:       "./data/repos",
			LFS:         "./data/lfs",
			Packages:    "./data/packages",
			Avatars:     "./data/avatars",
			Attachments: "./data/attachments",
		},
		Security: SecurityConfig{
			Password: PasswordConfig{
				Algorithm:   "argon2id",
				MemoryMB:    64,
				Iterations:  3,
				Parallelism: 2,
				SaltLength:  16,
				KeyLength:   32,
				MinLength:   12,
			},
			Login: LoginConfig{
				MaxAttempts:    5,
				LockoutMinutes: 15,
				WindowMinutes:  15,
			},
			Session: SessionConfig{
				TTLHours: 720,
			},
			PAT: PATConfig{
				AllowClassic:       true,
				AllowFineGrained:   true,
				DefaultScopesEmpty: true,
				MaxExpiryDays:      365,
			},
		},
		Modules: ModulesConfig{
			Core:          true,
			Collaboration: true,
			GitExt:        true,
			Registry:      true,
			CI:            false,
			IDE:           false,
			Ops:           true,
			Search:        "grep",
			Packages: PackagesModule{
				Enabled: []string{"generic", "npm", "docker", "go", "maven"},
			},
		},
		Limits: LimitsConfig{
			MaxGitConcurrency:     4,
			MaxWebhookConcurrency: 8,
			MaxCIConcurrency:      1,
			MemLimitMB:            256,
			Packages: PackageLimits{
				MaxConcurrent:          4,
				GenericMaxFileMB:       200,
				NpmMaxTarballMB:        50,
				NpmMaxMetadataKB:       1024,
				DockerMaxBlobMB:        500,
				DockerMaxManifestKB:    256,
				DockerUploadTTLMinutes: 60,
				GoMaxZipMB:             100,
				MavenMaxArtifactMB:     100,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Mail: MailConfig{
			Enabled: false,
			Host:    "smtp.example.com",
			Port:    587,
			From:    "noreply@example.com",
			TLS:     true,
		},
		Git: GitConfig{
			Bin:            "",
			TimeoutSeconds: 300,
			CheckVersion:   true,
			MinVersion:     "2.30",
		},
		SSH: SSHConfig{
			Mode:                    "builtin",
			HostKeyPath:             "./data/ssh_host_ed25519_key",
			HandshakeTimeoutSeconds: 10,
			IdleTimeoutSeconds:      300,
			MaxConnections:          32,
		},
		Webhook: WebhookConfig{
			TimeoutSeconds:        15,
			MaxRetries:            3,
			RetryIntervalSeconds:  60,
			BlockPrivateAddresses: true,
		},
		Backup: BackupConfig{
			Encrypt:          false,
			EncryptionKeyEnv: "GITLI_BACKUP_KEY",
			BackupDir:        "./data/backups",
		},
		OIDC: OIDCConfig{
			Enabled: false,
			Scopes:  []string{"openid", "profile", "email"},
		},
		LFS: LFSConfig{
			Enabled:     true,
			StoragePath: "./data/lfs",
			MaxFileMB:   5120,
		},
		CI: CIConfig{
			Enabled:              false,
			MaxConcurrency:       1,
			RunnerTimeoutSeconds: 3600,
			ArtifactMaxMB:        100,
		},
		Search: SearchConfig{
			Engine: "grep",
		},
		Runner: RunnerConfig{
			Enabled:                 true,
			JobTimeoutMinutes:       30,
			LogMaxMB:                10,
			HeartbeatTimeoutSeconds: 90,
			OfflineSweepIntervalSec: 60,
		},
		Audit: AuditConfig{
			RetentionDays:        365,
			ArchiveRetentionDays: 1095,
		},
	}
}
