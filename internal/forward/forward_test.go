package forward

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// serve 用 net.Pipe 模拟隧道流喂给 Handle，返回访客侧连接。
// deadline 防止实现回归导致测试挂死。
func serve(t *testing.T, target Target, rewriteHost bool) net.Conn {
	t.Helper()
	stream, mine := net.Pipe()
	go Handle(stream, target, rewriteHost)
	mine.SetDeadline(time.Now().Add(5 * time.Second)) //nolint:errcheck
	t.Cleanup(func() { mine.Close() })
	return mine
}

// TestHandleRewriteHost 校验 rewrite 模式：Host 改写为目标地址，path/query 与 body 原样透传。
func TestHandleRewriteHost(t *testing.T) {
	var gotHost, gotURI, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		gotURI = r.URL.RequestURI()
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		fmt.Fprintln(w, "ok")
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	mine := serve(t, Target{Addr: addr}, true)
	fmt.Fprintf(mine, "POST /api?q=1 HTTP/1.1\r\nHost: abc123.apps.soulink.cam\r\nContent-Length: 4\r\n\r\nbody")

	resp, err := http.ReadResponse(bufio.NewReader(mine), nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
		t.Fatalf("响应异常: %d %q", resp.StatusCode, body)
	}
	if gotHost != addr {
		t.Errorf("Host = %q, want 改写为目标 %q", gotHost, addr)
	}
	if gotURI != "/api?q=1" {
		t.Errorf("URI = %q, want /api?q=1", gotURI)
	}
	if gotBody != "body" {
		t.Errorf("body = %q, want body", gotBody)
	}
}

// TestHandleRawKeepsHost 校验默认透传模式：Host 原样转发（字节流不解析）。
func TestHandleRawKeepsHost(t *testing.T) {
	var gotHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		fmt.Fprintln(w, "ok")
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	mine := serve(t, Target{Addr: addr}, false)
	fmt.Fprintf(mine, "GET / HTTP/1.1\r\nHost: abc123.apps.soulink.cam\r\n\r\n")

	resp, err := http.ReadResponse(bufio.NewReader(mine), nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	resp.Body.Close()
	if gotHost != "abc123.apps.soulink.cam" {
		t.Errorf("Host = %q, want 原样透传公网域名", gotHost)
	}
}

// TestHandleRewriteUpgrade 校验 rewrite 模式遇到 Upgrade 请求：
// 改写 Host 转发后，升级后的字节流恢复双向透传。
func TestHandleRewriteUpgrade(t *testing.T) {
	// 裸 TCP 目标：读请求头，回 101，随后 echo 所有字节（模拟 WebSocket）
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	head := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		br := bufio.NewReader(c)
		var sb strings.Builder
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			sb.WriteString(line)
			if line == "\r\n" {
				break
			}
		}
		head <- sb.String()
		c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n\r\n")) //nolint:errcheck
		io.Copy(c, br)                                            //nolint:errcheck // echo 升级后的字节
	}()

	mine := serve(t, Target{Addr: ln.Addr().String()}, true)
	fmt.Fprintf(mine, "GET /ws HTTP/1.1\r\nHost: abc123.apps.soulink.cam\r\nConnection: upgrade\r\nUpgrade: websocket\r\n\r\n")

	br := bufio.NewReader(mine)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 101 {
		t.Fatalf("状态码 = %d, want 101", resp.StatusCode)
	}

	reqHead := <-head
	if !strings.Contains(reqHead, "Host: "+ln.Addr().String()) {
		t.Errorf("升级请求 Host 未改写:\n%s", reqHead)
	}

	// 升级后的帧必须原样透传
	mine.Write([]byte("ping-raw")) //nolint:errcheck
	buf := make([]byte, 8)
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatalf("读取透传帧: %v", err)
	}
	if string(buf) != "ping-raw" {
		t.Errorf("透传帧 = %q, want ping-raw", buf)
	}
}
