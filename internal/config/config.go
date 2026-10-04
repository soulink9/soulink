// Package config 管理 local 的用户级配置（~/.soulink/config.yaml）。
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 是 local 持久化的用户配置。
type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
}

// Path 返回配置文件路径：~/.soulink/config.yaml。
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".soulink", "config.yaml"), nil
}

// Load 读取配置；文件不存在时返回空配置（非错误）。
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	return cfg, yaml.Unmarshal(data, cfg)
}

// Save 写配置：目录 0700、文件 0600（含 token）。
func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
