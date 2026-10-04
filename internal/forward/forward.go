// Package forward 把隧道 stream 转发到本地目标端口。
// 隧道上传输的是原始 HTTP/1.1 字节流，本包不做任何协议解析，
// 因此 WebSocket 升级流量天然透明。
package forward

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

// Target 是归一化后的转发目标。
type Target struct {
	Addr string // host:port 拨号地址
	TLS  bool   // https 目标：TLS 拨号（SNI = host）
}

// dialTimeout 本地目标拨号超时。
const dialTimeout = 10 * time.Second

// badGateway 是本地目标不可达时回写的最小合法 HTTP 响应。
// hub 侧无需特殊处理，访客看到的就是标准 502。
const badGateway = "HTTP/1.1 502 Bad Gateway\r\n" +
	"Content-Type: text/plain; charset=utf-8\r\n" +
	"Content-Length: 25\r\n" +
	"Connection: close\r\n" +
	"\r\nsoulink: local unreachable"

// dial 按 Target 拨号：TLS 目标用 SNI 握手，否则纯 TCP。
func dial(t Target) (net.Conn, error) {
	d := &net.Dialer{Timeout: dialTimeout}
	if !t.TLS {
		return d.Dial("tcp", t.Addr)
	}
	host, _, err := net.SplitHostPort(t.Addr)
	if err != nil {
		return nil, err
	}
	return tls.DialWithDialer(d, "tcp", t.Addr, &tls.Config{ServerName: host})
}

// Handle 把一条 yamux stream 与目标连接双向桥接，直到任一端关闭。
func Handle(stream net.Conn, target Target) {
	defer stream.Close()

	lc, err := dial(target)
	if err != nil {
		stream.Write([]byte(badGateway))
		return
	}
	defer lc.Close()

	// 任一方向拷贝结束即整体关闭，避免半开泄漏
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(lc, stream)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(stream, lc)
		done <- struct{}{}
	}()
	<-done
}
