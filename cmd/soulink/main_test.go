package main

import (
	"testing"

	"github.com/soulink/soulink-local/client"
	"github.com/soulink/soulink-local/internal/config"
)

// TestResolveDefaultsPrecedence 校验 flag > 配置文件 的取值顺序。
func TestResolveDefaultsPrecedence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := config.Save(&config.Config{Server: "https://cfg.example.com", Token: "cfg-token"}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	cases := []struct {
		name       string
		opts       client.Options
		wantServer string
		wantToken  string
	}{
		{
			name:       "flag 优先于配置文件",
			opts:       client.Options{Server: "https://flag.example.com", Token: "flag-token"},
			wantServer: "https://flag.example.com",
			wantToken:  "flag-token",
		},
		{
			name:       "无 flag 时回落配置文件",
			opts:       client.Options{},
			wantServer: "https://cfg.example.com",
			wantToken:  "cfg-token",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			resolveDefaults(&opts)
			if opts.Server != tc.wantServer {
				t.Errorf("Server = %q, want %q", opts.Server, tc.wantServer)
			}
			if opts.Token != tc.wantToken {
				t.Errorf("Token = %q, want %q", opts.Token, tc.wantToken)
			}
		})
	}
}

// TestResolveDefaultsBuiltinFallback 校验无配置无 flag 时落到内置默认。
func TestResolveDefaultsBuiltinFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // 无 ~/.soulink/config.yaml

	opts := client.Options{}
	resolveDefaults(&opts)

	if opts.Server != defaultServer {
		t.Errorf("Server = %q, want 内置默认 %q", opts.Server, defaultServer)
	}
	if opts.Token != defaultGuestToken {
		t.Errorf("Token = %q, want guest %q", opts.Token, defaultGuestToken)
	}
}

// TestBuiltinDefaultsNonEmpty 守住官方构建的前提：内置 server 必须非空且带 scheme。
// 裸 host 会被 client 补成 ws://（明文），故此处强校验 https:// 前缀。
func TestBuiltinDefaultsNonEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if defaultServer == "" {
		t.Fatal("defaultServer 为空：官方构建必须带内置默认，仅自托管构建可用 -X main.defaultServer= 清空")
	}
	if len(defaultServer) < len("https://") || defaultServer[:len("https://")] != "https://" {
		t.Errorf("defaultServer = %q, 必须带 https:// 前缀", defaultServer)
	}
}
