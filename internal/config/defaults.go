// SPDX-License-Identifier: AGPL-3.0
package config

// Defaults returns a configuration populated with safe defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			HTTP:        ":8080",
			SSH:         ":2222",
			ExternalURL: "http://localhost:8080",
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
			},
			Login: LoginConfig{
				MaxAttempts:    5,
				LockoutMinutes: 15,
				WindowMinutes:  15,
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
	}
}
