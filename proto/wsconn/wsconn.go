// Package wsconn 把 *websocket.Conn 适配成 net.Conn，供 yamux 在其上建会话。
// 握手阶段（注册 JSON）直接使用 websocket 消息，握手完成后才进入本适配层。
package wsconn

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Conn 在单条 websocket 连接上提供面向字节流的 net.Conn 语义。
// yamux 只有一个读循环，但可能有多个并发写者，因此写侧加互斥锁。
type Conn struct {
	c  *websocket.Conn
	r  io.Reader // 当前未读完的二进制帧
	wm sync.Mutex
}

// New 包装一条已完成协议升级的 websocket 连接。
func New(c *websocket.Conn) *Conn {
	return &Conn{c: c}
}

func (w *Conn) Read(p []byte) (int, error) {
	for {
		if w.r == nil {
			mt, r, err := w.c.NextReader()
			if err != nil {
				return 0, err
			}
			if mt != websocket.BinaryMessage {
				// yamux 阶段不应再出现文本帧/控制帧以外的数据，跳过非二进制帧
				continue
			}
			w.r = r
		}
		n, err := w.r.Read(p)
		if err == io.EOF {
			w.r = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (w *Conn) Write(p []byte) (int, error) {
	w.wm.Lock()
	defer w.wm.Unlock()
	wr, err := w.c.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	n, err := wr.Write(p)
	if cerr := wr.Close(); err == nil {
		err = cerr
	}
	return n, err
}

func (w *Conn) Close() error                       { return w.c.Close() }
func (w *Conn) LocalAddr() net.Addr                { return w.c.LocalAddr() }
func (w *Conn) RemoteAddr() net.Addr               { return w.c.RemoteAddr() }
func (w *Conn) SetReadDeadline(t time.Time) error  { return w.c.SetReadDeadline(t) }
func (w *Conn) SetWriteDeadline(t time.Time) error { return w.c.SetWriteDeadline(t) }

func (w *Conn) SetDeadline(t time.Time) error {
	if err := w.SetReadDeadline(t); err != nil {
		return err
	}
	return w.SetWriteDeadline(t)
}
