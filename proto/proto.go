// Package proto 定义 soulink hub 与 local 之间的隧道握手协议。
// 约束：本包只放协议 DTO 与常量，不依赖 local/hub 的任何内部包。
package proto

// ConnectPath 是 local 与 hub 建立隧道（wss 升级）的固定路径。
const ConnectPath = "/_soulink/v1/connect"

// 握手消息类型
const (
	MsgTypeRegister   = "register"
	MsgTypeRegistered = "registered"
	MsgTypeError      = "error"
)

// 错误码
const (
	ErrUnauthorized      = "unauthorized"
	ErrSubdomainTaken    = "subdomain_taken"
	ErrSubdomainInvalid  = "subdomain_invalid"
	ErrSubdomainForbid   = "subdomain_forbidden"
	ErrBadRequest        = "bad_request"
)

// Register 是 wss 升级后 local 发送的第一条消息（JSON 文本帧）。
type Register struct {
	Type               string `json:"type"` // 固定 "register"
	Version            int    `json:"version"`
	ClientName         string `json:"client_name"`
	RequestedSubdomain string `json:"requested_subdomain"` // 空 = 由 hub 随机分配
}

// Registered 是 hub 对 Register 的成功应答，此后连接进入 yamux 数据面。
type Registered struct {
	Type     string   `json:"type"` // 固定 "registered"
	TunnelID string   `json:"tunnel_id"`
	URLs     []string `json:"urls"`
	Tier     string   `json:"tier"`
}

// Error 是 hub 的失败应答。
type Error struct {
	Type    string `json:"type"` // 固定 "error"
	Code    string `json:"code"`
	Message string `json:"message"`
}
