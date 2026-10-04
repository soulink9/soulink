package client

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	cases := []struct {
		raw     string
		addr    string
		tls     bool
		wantErr bool
	}{
		{"3000", "127.0.0.1:3000", false, false},
		{" 3000 ", "127.0.0.1:3000", false, false},
		{"localhost:3000", "localhost:3000", false, false},
		{"127.0.0.1:8080", "127.0.0.1:8080", false, false},
		{"192.168.1.10:8080", "192.168.1.10:8080", false, false},
		{"example.com:9000", "example.com:9000", false, false},
		{"http://example.com", "example.com:80", false, false},
		{"http://example.com:8080", "example.com:8080", false, false},
		{"http://192.168.1.10:8080", "192.168.1.10:8080", false, false},
		{"https://www.baidu.com", "www.baidu.com:443", true, false},
		{"https://example.com:8443", "example.com:8443", true, false},
		{"https://example.com/", "example.com:443", true, false},

		// 非法输入
		{"", "", false, true},
		{"http://example.com/api", "", false, true},  // 带路径
		{"https://example.com?x=1", "", false, true}, // 带查询
		{"ftp://example.com", "", false, true},       // 不支持的 scheme
		{"http://", "", false, true},                 // 缺 host
		{"not-a-port-or-host", "", false, true},      // 无法识别
	}
	for _, c := range cases {
		got, err := ParseTarget(c.raw)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseTarget(%q) = %+v, want error", c.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseTarget(%q) error: %v", c.raw, err)
			continue
		}
		if got.Addr != c.addr || got.TLS != c.tls {
			t.Errorf("ParseTarget(%q) = {%s tls=%v}, want {%s tls=%v}", c.raw, got.Addr, got.TLS, c.addr, c.tls)
		}
	}
}
