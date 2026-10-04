// Package client 负责与 hub 建立并维持隧道连接。
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	Target    string // 本地目标，host:port
	Insecure  bool   // wss 跳过证书校验（自签场景）
}

const (
	handshakeTimeout = 10 * time.Second
	maxBackoff       = 30 * time.Second
)

// Run 维持隧道长连接：断线后指数退避重连，直到 ctx 取消。
func Run(ctx context.Context, opts Options) error {
	backoff := time.Second
	for {
		err := runOnce(ctx, opts)
		if ctx.Err() != nil {
			return ctx.Err()
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

func runOnce(ctx context.Context, opts Options) error {
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
	ws, _, err := dialer.DialContext(ctx, u, header)
	if err != nil {
		return fmt.Errorf("dial hub: %w", err)
	}
	defer ws.Close()

	// ---- 握手阶段：一条 JSON 文本帧请求，一条应答 ----
	reg := proto.Register{
		Type:               proto.MsgTypeRegister,
		Version:            1,
		ClientName:         clientName(),
		RequestedSubdomain: opts.Subdomain,
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
	log.Printf("tunnel %s online (%s), forwarding to %s", reply.TunnelID, reply.Tier, opts.Target)

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
		go forward.Handle(stream, opts.Target)
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

// NormalizeTarget 把 "3000" 之类的裸端口补全为 "127.0.0.1:3000"。
func NormalizeTarget(target string) string {
	if _, _, err := net.SplitHostPort(target); err == nil {
		return target
	}
	if _, err := net.LookupPort("tcp", target); err == nil {
		return net.JoinHostPort("127.0.0.1", target)
	}
	return target
}
