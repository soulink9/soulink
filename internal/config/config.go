// Package config 管理 local 的用户级配置（~/.soulink/config.yaml）。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 是 local 持久化的用户配置。
type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
}

// ValidKeys 是 config set/unset 可管理的键（与 Config 字段一一对应）。
var ValidKeys = []string{"server", "token"}

func validKey(key string) bool {
	for _, k := range ValidKeys {
		if k == key {
			return true
		}
	}
	return false
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
// 文件存在但损坏/不可读时返回带路径的错误，调用方应显式呈现而非静默忽略。
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
		return nil, fmt.Errorf("读取配置文件 %s: %w", p, err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s: %w（请修复或删除该文件）", p, err)
	}
	return cfg, nil
}

// Save 整体写配置：目录 0700、文件 0600（含 token）。
// 注意会丢弃文件中的注释与未知字段；单键修改请用 Set/Unset。
func Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return writeFile(data)
}

// Set 更新单个配置键（读-改-写经 yaml.Node，保留注释与未知字段）。
func Set(key, value string) error {
	if !validKey(key) {
		return fmt.Errorf("未知配置键 %q（可用：%s）", key, strings.Join(ValidKeys, ", "))
	}
	return update(func(m *yaml.Node) { nodeSet(m, key, value) })
}

// Unset 删除单个配置键；键不存在时为无操作。
func Unset(key string) error {
	if !validKey(key) {
		return fmt.Errorf("未知配置键 %q（可用：%s）", key, strings.Join(ValidKeys, ", "))
	}
	return update(func(m *yaml.Node) { nodeUnset(m, key) })
}

// update 读-改-写配置文件的顶层 mapping；文件不存在时从空 mapping 起步。
func update(mut func(m *yaml.Node)) error {
	p, err := Path()
	if err != nil {
		return err
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode}
	data, err := os.ReadFile(p)
	switch {
	case os.IsNotExist(err):
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	case err != nil:
		return fmt.Errorf("读取配置文件 %s: %w", p, err)
	default:
		if err := yaml.Unmarshal(data, doc); err != nil {
			return fmt.Errorf("解析配置文件 %s: %w（请修复或删除该文件）", p, err)
		}
		if len(doc.Content) == 0 {
			doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		}
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return fmt.Errorf("配置文件 %s 顶层必须是键值映射（请修复或删除该文件）", p)
	}
	mut(m)
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return writeFile(out)
}

func writeFile(data []byte) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return err
	}
	// WriteFile 的权限只在创建时生效；文件被手工建过（umask 0644）时也要压回 0600（含 token）
	return os.Chmod(p, 0o600)
}

// nodeSet 在 mapping 上设置键值；已存在则原地改值（保留该键上的注释）。
func nodeSet(m *yaml.Node, key, value string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			v := m.Content[i+1]
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func nodeUnset(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}
