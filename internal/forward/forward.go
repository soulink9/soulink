// Package forward 把隧道 stream 转发到本地目标端口。
//
// 默认是原始 HTTP/1.1 字节流透传，不做任何协议解析，WebSocket 升级流量
// 天然透明。rewriteHost 模式逐请求解析 HTTP 头并改写 Host（解决本地服务
// 只接受回环 Host 的白名单校验），WebSocket 升级请求转发后恢复透传。
package forward

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
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
func Handle(stream net.Conn, target Target, rewriteHost bool) {
	defer stream.Close()

	lc, err := dial(target)
	if err != nil {
		// 访客会收到 502；local 端必须留痕，否则无法区分「目标挂了」与「隧道断了」
		log.Printf("目标 %s 拨号失败（访客已回 502）: %v", target.Addr, err)
		stream.Write([]byte(badGateway))
		return
	}
	defer lc.Close()

	if rewriteHost {
		bridgeRewriteHost(lc, stream, target.Addr)
		return
	}
	bridge(lc, stream)
}

// bridge 原始字节流双向桥接：任一方向拷贝结束即整体关闭，避免半开泄漏。
func bridge(lc, stream net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(lc, stream) //nolint:errcheck
		done <- struct{}{}
	}()
	go func() {
		io.Copy(stream, lc) //nolint:errcheck
		done <- struct{}{}
	}()
	<-done
}

// bridgeRewriteHost 请求方向逐请求改写 Host 为目标地址；响应方向始终透传。
// hub 每条 stream 只发一个请求（DisableKeepAlives），循环写法对将来的
// keep-alive 同样安全；Upgrade 请求（WebSocket）转发后恢复字节流透传。
func bridgeRewriteHost(lc, stream net.Conn, targetAddr string) {
	br := bufio.NewReader(stream)
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(stream, lc) //nolint:errcheck
		done <- struct{}{}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			req, err := http.ReadRequest(br)
			if err != nil {
				return // 对端关闭（EOF）或协议异常，结束转发
			}
			req.RequestURI = "" // Request.Write 要求客户端语义，清空后由 URL 重建 origin-form
			req.Host = targetAddr
			if err := req.Write(lc); err != nil {
				return
			}
			if isUpgrade(req) {
				io.Copy(lc, br) //nolint:errcheck // WebSocket 后续帧透传（br 可能已缓冲）
				return
			}
		}
	}()
	<-done
}

// isUpgrade 判定 WebSocket 等协议升级请求（Connection: upgrade + Upgrade 头）。
func isUpgrade(req *http.Request) bool {
	return req.Header.Get("Upgrade") != "" &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}
