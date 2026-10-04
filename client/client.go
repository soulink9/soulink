// Package client 负责与 hub 建立并维持隧道连接。
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/soulink/soulink-local/internal/forward"
	"github.com/soulink/soulink-local/proto"
	"github.com/soulink/soulink-local/proto/wsconn"
)

// Options 是一次隧道会话的全部参数。
type Options struct {
	Server    string // hub 地址，如 "localhost:8080" 或 "wss://hub.example.com"
	Token     string
	Subdomain string // 空 = 随机分配
	Target    string // 转发目标：端口 / host:port / http(s)://host[:port]，见 ParseTarget
	Insecure  bool   // wss 跳过证书校验（自签场景）
}

const (
	handshakeTimeout = 10 * time.Second
	maxBackoff       = 30 * time.Second
	// stableThreshold：runOnce 存活超过该时长视为一波健康连接，
	// 断开后退避重置。短于此值的失败（握手即败/秒断）不重置，
	// 保留对抖动 hub 的退避保护。
	stableThreshold = 30 * time.Second
)

// errUnauthorized 表示 hub 拒绝了凭据（401）。
// 重试不会改变结果，Run 遇此错误直接退出。
var errUnauthorized = errors.New("unauthorized: token 无效或已吊销（检查 `soulink login` 或 --token）")

// Run 维持隧道长连接：断线后指数退避重连，直到 ctx 取消。
// target 解析失败、认证被拒（401）属配置错误，立即返回不重试。
func Run(ctx context.Context, opts Options) error {
	target, err := ParseTarget(opts.Target)
	if err != nil {
		return err
	}
	probeTarget(target)
	backoff := time.Second
	for {
		start := time.Now()
		err := runOnce(ctx, opts, target)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, errUnauthorized) {
			return err // 重试不会改变凭据
		}
		if time.Since(start) >= stableThreshold {
			// 健康连接后的断开视为新事件：从历史最大退避重新开始，
			// 避免数小时前的失败风暴永久抬高重连延迟
			backoff = time.Second
		}
		log.Printf("tunnel closed: %v; reconnecting in %s", err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// probeTarget 启动期一次性探测目标可达性，仅告警不阻断：
// 目标服务生命周期独立于隧道（可能晚启动/重启中），隧道仍正常建立，
// 但用户应第一时间知道「现在访问会是 502」。
func probeTarget(t forward.Target) {
	c, err := net.DialTimeout("tcp", t.Addr, 2*time.Second)
	if err != nil {
		log.Printf("⚠ 目标 %s 当前不可达（%v）；隧道将正常建立，访客在目标就绪前会看到 502", t.Addr, err)
		return
	}
	c.Close()
}

func runOnce(ctx context.Context, opts Options, target forward.Target) error {
	u, err := connectURL(opts.Server)
	if err != nil {
		return err
	}

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = handshakeTimeout
	if opts.Insecure {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // 用户显式指定
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+opts.Token)
	ws, resp, err := dialer.DialContext(ctx, u, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return errUnauthorized
		}
		return fmt.Errorf("dial hub: %w", err)
	}
	defer ws.Close()

	// ---- 握手阶段：一条 JSON 文本帧请求，一条应答 ----
	reg := proto.Register{
		Type:               proto.MsgTypeRegister,
		Version:            1,
		ClientName:         clientName(),
		RequestedSubdomain: opts.Subdomain,
		Target:             target.Addr,
	}
	ws.SetWriteDeadline(time.Now().Add(handshakeTimeout))
	if err := ws.WriteJSON(reg); err != nil {
		return fmt.Errorf("send register: %w", err)
	}
	ws.SetReadDeadline(time.Now().Add(handshakeTimeout))
	_, raw, err := ws.ReadMessage()
	if err != nil {
		return fmt.Errorf("read register reply: %w", err)
	}
	ws.SetReadDeadline(time.Time{})
	ws.SetWriteDeadline(time.Time{})

	var reply proto.Registered
	if err := json.Unmarshal(raw, &reply); err != nil {
		return fmt.Errorf("parse register reply: %w", err)
	}
	if reply.Type == proto.MsgTypeError {
		var e proto.Error
		json.Unmarshal(raw, &e)
		return fmt.Errorf("hub rejected: %s (%s)", e.Code, e.Message)
	}
	if reply.Type != proto.MsgTypeRegistered {
		return fmt.Errorf("unexpected reply type %q", reply.Type)
	}
	for _, u := range reply.URLs {
		fmt.Printf("→ %s\n", u)
	}
	if reply.TunnelTTLSec > 0 {
		ttl := time.Duration(reply.TunnelTTLSec) * time.Second
		fmt.Printf("⚠ 限时通道：隧道将于 %s 后自动回收（注册用户可获得不限时隧道）\n", ttl)
	}
	log.Printf("tunnel %s online (%s), forwarding to %s", reply.TunnelID, reply.Tier, target.Addr)

	// ---- 数据面：yamux 会话 ----
	sess, err := yamux.Client(wsconn.New(ws), nil)
	if err != nil {
		return fmt.Errorf("yamux client: %w", err)
	}
	defer sess.Close()

	// ctx 取消时主动关闭会话以退出 Accept 循环
	go func() {
		select {
		case <-ctx.Done():
			sess.Close()
		case <-sess.CloseChan():
		}
	}()

	for {
		stream, err := sess.AcceptStream()
		if err != nil {
			return fmt.Errorf("accept stream: %w", err)
		}
		go forward.Handle(stream, target)
	}
}

// connectURL 把用户输入的 server 规格化为完整的 wss 连接地址。
// 注意：裸 "localhost:8080" 会被 url.Parse 误判 scheme，必须先补 "://"。
func connectURL(server string) (string, error) {
	if server == "" {
		return "", fmt.Errorf("server is required")
	}
	if !strings.Contains(server, "://") {
		server = "ws://" + server
	}
	u, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	u.Path = proto.ConnectPath
	return u.String(), nil
}

func clientName() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}

// ParseTarget 把用户输入归一化为转发目标。支持的输入形式：
//
//	3000                       → 127.0.0.1:3000        (TCP)
//	http                       → 127.0.0.1:80          (TCP，服务名按 /etc/services 解析)
//	localhost:3000             → localhost:3000        (TCP)
//	192.168.1.10:8080          → 192.168.1.10:8080     (TCP)
//	http://example.com         → example.com:80        (TCP)
//	http://example.com:8080    → example.com:8080      (TCP)
//	https://www.baidu.com      → www.baidu.com:443     (TLS，SNI=host)
//	https://example.com:8443   → example.com:8443      (TLS)
//
// URL 带路径/查询（如 http://host/api）属于误用，直接报错；
// 其他无法识别的输入同样报错，由调用方在启动期退出。
func ParseTarget(raw string) (forward.Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return forward.Target{}, fmt.Errorf("target is required")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return forward.Target{}, fmt.Errorf("invalid target %q: %w", raw, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return forward.Target{}, fmt.Errorf("unsupported target scheme %q (want http or https)", u.Scheme)
		}
		if u.Path != "" && u.Path != "/" || u.RawQuery != "" {
			return forward.Target{}, fmt.Errorf("target %q must not contain a path or query — soulink forwards the whole origin", raw)
		}
		host := u.Hostname()
		if host == "" {
			return forward.Target{}, fmt.Errorf("invalid target %q: missing host", raw)
		}
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		return forward.Target{Addr: net.JoinHostPort(host, port), TLS: u.Scheme == "https"}, nil
	}

	// 已是 host:port 形式
	if _, _, err := net.SplitHostPort(raw); err == nil {
		return forward.Target{Addr: raw}, nil
	}
	// 裸端口或服务名 → 回环
	if _, err := net.LookupPort("tcp", raw); err == nil {
		return forward.Target{Addr: net.JoinHostPort("127.0.0.1", raw)}, nil
	}
	return forward.Target{}, fmt.Errorf("invalid target %q: want a port (3000), host:port, or http(s)://host[:port]", raw)
}
