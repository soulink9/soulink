package main

import (
	"os"
	"testing"

	"github.com/soulink/soulink-local/client"
	"github.com/soulink/soulink-local/internal/config"
)

// isolateEnv 把两个 env 覆盖层置空，避免宿主机环境污染用例。
func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envServer, "")
	t.Setenv(envToken, "")
}

// TestResolveDefaultsPrecedence 校验 flag > env > 配置文件 > 内置默认 的取值顺序。
func TestResolveDefaultsPrecedence(t *testing.T) {
	isolateEnv(t)
	t.Setenv("HOME", t.TempDir())
	if err := config.Save(&config.Config{Server: "https://cfg.example.com", Token: "cfg-token"}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	cases := []struct {
		name       string
		opts       client.Options
		envServer  string
		envToken   string
		wantServer string
		wantToken  string
	}{
		{
			name:       "flag 优先于一切",
			opts:       client.Options{Server: "https://flag.example.com", Token: "flag-token"},
			envServer:  "https://env.example.com",
			envToken:   "env-token",
			wantServer: "https://flag.example.com",
			wantToken:  "flag-token",
		},
		{
			name:       "env 优先于配置文件",
			envServer:  "https://env.example.com",
			envToken:   "env-token",
			wantServer: "https://env.example.com",
			wantToken:  "env-token",
		},
		{
			name:       "无 flag 无 env 时回落配置文件",
			wantServer: "https://cfg.example.com",
			wantToken:  "cfg-token",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envServer, tc.envServer)
			t.Setenv(envToken, tc.envToken)
			opts := tc.opts
			if err := resolveDefaults(&opts); err != nil {
				t.Fatalf("resolveDefaults: %v", err)
			}
			if opts.Server != tc.wantServer {
				t.Errorf("Server = %q, want %q", opts.Server, tc.wantServer)
			}
			if opts.Token != tc.wantToken {
				t.Errorf("Token = %q, want %q", opts.Token, tc.wantToken)
			}
		})
	}
}

// TestResolveDefaultsBuiltinFallback 校验无配置无 flag 无 env 时落到内置默认与 guest。
func TestResolveDefaultsBuiltinFallback(t *testing.T) {
	isolateEnv(t)
	t.Setenv("HOME", t.TempDir()) // 无 ~/.soulink/config.yaml

	opts := client.Options{}
	if err := resolveDefaults(&opts); err != nil {
		t.Fatalf("resolveDefaults: %v", err)
	}
	if opts.Server != defaultServer {
		t.Errorf("Server = %q, want 内置默认 %q", opts.Server, defaultServer)
	}
	if opts.Token != defaultGuestToken {
		t.Errorf("Token = %q, want guest %q", opts.Token, defaultGuestToken)
	}
}

// TestResolveDefaultsCorruptConfig 配置文件损坏时必须显式报错，不得静默回落默认值。
func TestResolveDefaultsCorruptConfig(t *testing.T) {
	isolateEnv(t)
	t.Setenv("HOME", t.TempDir())
	if err := config.Set("server", "https://ok.example.com"); err != nil {
		t.Fatal(err)
	}
	p, _ := config.Path()
	if err := os.WriteFile(p, []byte("server: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := client.Options{}
	if err := resolveDefaults(&opts); err == nil {
		t.Fatal("配置文件损坏时 resolveDefaults 应报错")
	}
}

// TestBuiltinDefaultsNonEmpty 守住官方构建的前提：内置 server 必须非空且带 scheme。
// 裸 host 会被 client 补成 ws://（明文），故此处强校验 https:// 前缀。
func TestBuiltinDefaultsNonEmpty(t *testing.T) {
	isolateEnv(t)
	if defaultServer == "" {
		t.Fatal("defaultServer 为空：官方构建必须带内置默认，仅自托管构建可用 -X main.defaultServer= 清空")
	}
	if len(defaultServer) < len("https://") || defaultServer[:len("https://")] != "https://" {
		t.Errorf("defaultServer = %q, 必须带 https:// 前缀", defaultServer)
	}
}

// TestMaskToken token 打码：长 token 保留首尾，短 token 整体打码。
func TestMaskToken(t *testing.T) {
	if got := maskToken("slk_abcdefgh1234"); got != "slk_…1234" {
		t.Errorf("maskToken = %q", got)
	}
	if got := maskToken("short"); got != "****" {
		t.Errorf("maskToken(short) = %q", got)
	}
}
