package config

import (
	"os"
	"strings"
	"testing"
)

// TestSetUnsetRoundTrip 校验单键写入/更新/删除，以及未知字段与注释的保留。
func TestSetUnsetRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// 预置一份含注释与未知字段的配置
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(strings.TrimSuffix(p, "/config.yaml"), 0o700); err != nil {
		t.Fatal(err)
	}
	raw := "# 我的 soulink 配置\nserver: https://old.example.com # 行尾注释\nextra: keep-me\n"
	if err := os.WriteFile(p, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	// 更新已有键：值变化，注释与未知字段保留
	if err := Set("server", "https://new.example.com"); err != nil {
		t.Fatalf("Set server: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server != "https://new.example.com" {
		t.Errorf("Server = %q", cfg.Server)
	}
	data, _ := os.ReadFile(p)
	for _, want := range []string{"# 我的 soulink 配置", "extra: keep-me"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("写回后丢失 %q；内容：\n%s", want, data)
		}
	}

	// 追加新键
	if err := Set("token", "tok-123456"); err != nil {
		t.Fatalf("Set token: %v", err)
	}
	cfg, _ = Load()
	if cfg.Token != "tok-123456" {
		t.Errorf("Token = %q", cfg.Token)
	}

	// 删除键；再次删除为无操作
	if err := Unset("server"); err != nil {
		t.Fatalf("Unset: %v", err)
	}
	if err := Unset("server"); err != nil {
		t.Fatalf("Unset again: %v", err)
	}
	cfg, _ = Load()
	if cfg.Server != "" || cfg.Token != "tok-123456" {
		t.Errorf("Unset 后 Server=%q Token=%q", cfg.Server, cfg.Token)
	}
}

// TestSetInvalidKey 拒绝未知键。
func TestSetInvalidKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Set("nope", "x"); err == nil {
		t.Fatal("Set(未知键) 应报错")
	}
	if err := Unset("nope"); err == nil {
		t.Fatal("Unset(未知键) 应报错")
	}
}

// TestSetOnMissingFile 配置文件不存在时 Set 从零创建，权限 0600。
func TestSetOnMissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Set("token", "abc"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	cfg, err := Load()
	if err != nil || cfg.Token != "abc" {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
	p, _ := Path()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("权限 = %o，want 600", fi.Mode().Perm())
	}
}

// TestLoadCorrupt 损坏的配置文件必须显式报错（含路径），不得静默吞掉。
func TestLoadCorrupt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Set("server", "https://ok.example.com"); err != nil {
		t.Fatal(err)
	}
	p, _ := Path()
	if err := os.WriteFile(p, []byte("server: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("损坏文件应报错")
	}
	if !strings.Contains(err.Error(), p) {
		t.Errorf("错误应包含路径 %s: %v", p, err)
	}
	if err := Set("token", "x"); err == nil {
		t.Fatal("损坏文件上 Set 应报错而非覆盖")
	}
}
