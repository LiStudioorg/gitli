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
}

func (c *Config) validate() error {
	if c.App.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if c.Server.HTTPAddr == "" {
		return fmt.Errorf("http_addr is required")
	}
	return nil
}

func (c *Config) DBPath() string {
	return filepath.Join(c.App.DataDir, "gitli.db")
}

func (c *Config) ReposDir() string {
	return filepath.Join(c.App.DataDir, "repos")
}
