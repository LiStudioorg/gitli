package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server ServerConfig `toml:"server"`
	App    AppConfig    `toml:"app"`
	Log    LogConfig    `toml:"log"`
	OAuth2 OAuth2Config `toml:"oauth2"`
}

type ServerConfig struct {
	HTTPAddr string `toml:"http_addr"`
	SSHAddr  string `toml:"ssh_addr"`
	GitAddr  string `toml:"git_addr"`
	RootURL  string `toml:"root_url"`
}

type AppConfig struct {
	DataDir string `toml:"data_dir"`
}

type LogConfig struct {
	Level string `toml:"level"`
}

// OAuth2Config 通用 OAuth2/OIDC 授权码流程配置。
type OAuth2Config struct {
	Enabled      bool     `toml:"enabled"`
	ClientID     string   `toml:"client_id"`
	ClientSecret string   `toml:"client_secret"`
	AuthURL      string   `toml:"auth_url"`
	TokenURL     string   `toml:"token_url"`
	UserinfoURL  string   `toml:"userinfo_url"`
	Scopes       []string `toml:"scopes"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPAddr: ":3000",
			SSHAddr:  ":2222",
			GitAddr:  ":9418",
			RootURL:  "http://localhost:3000",
		},
		App: AppConfig{
			DataDir: "./data",
		},
		Log: LogConfig{
			Level: "info",
		},
		OAuth2: OAuth2Config{
			Scopes: []string{"openid", "profile", "email"},
		},
	}
}

// Load 从 path 读取 TOML 配置（文件不存在则用默认值 + 环境变量）。
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
		}
	}
	cfg.applyEnv()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) applyEnv() {
	if v := os.Getenv("GITLI_HTTP_ADDR"); v != "" {
		c.Server.HTTPAddr = v
	}
	if v := os.Getenv("GITLI_SSH_ADDR"); v != "" {
		c.Server.SSHAddr = v
	}
	if v := os.Getenv("GITLI_GIT_ADDR"); v != "" {
		c.Server.GitAddr = v
	}
	if v := os.Getenv("GITLI_ROOT_URL"); v != "" {
		c.Server.RootURL = v
	}
	if v := os.Getenv("GITLI_DATA_DIR"); v != "" {
		c.App.DataDir = v
	}
	if v := os.Getenv("GITLI_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	// OAuth2 环境变量覆盖
	if v := os.Getenv("GITLI_OAUTH_ENABLED"); v == "1" || v == "true" || v == "yes" {
		c.OAuth2.Enabled = true
	}
	if v := os.Getenv("GITLI_OAUTH_CLIENT_ID"); v != "" {
		c.OAuth2.ClientID = v
	}
	if v := os.Getenv("GITLI_OAUTH_CLIENT_SECRET"); v != "" {
		c.OAuth2.ClientSecret = v
	}
	if v := os.Getenv("GITLI_OAUTH_AUTH_URL"); v != "" {
		c.OAuth2.AuthURL = v
	}
	if v := os.Getenv("GITLI_OAUTH_TOKEN_URL"); v != "" {
		c.OAuth2.TokenURL = v
	}
	if v := os.Getenv("GITLI_OAUTH_USERINFO_URL"); v != "" {
		c.OAuth2.UserinfoURL = v
	}
}

func (c *Config) validate() error {
	if c.App.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if c.Server.HTTPAddr == "" {
		return fmt.Errorf("http_addr is required")
	}
	// git http-backend 等 CGI 子进程依赖绝对路径，这里统一转绝对
	abs, err := filepath.Abs(c.App.DataDir)
	if err != nil {
		return fmt.Errorf("resolve data_dir: %w", err)
	}
	c.App.DataDir = abs
	return nil
}

func (c *Config) DBPath() string {
	return filepath.Join(c.App.DataDir, "gitli.db")
}

func (c *Config) ReposDir() string {
	return filepath.Join(c.App.DataDir, "repos")
}
