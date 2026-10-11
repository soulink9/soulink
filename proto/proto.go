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
	ErrUnauthorized    = "unauthorized"
	ErrSubdomainTaken  = "subdomain_taken"
	ErrSubdomainInvalid = "subdomain_invalid"
	ErrSubdomainForbid = "subdomain_forbidden"
	ErrTunnelLimit     = "tunnel_limit"
	ErrTargetForbid    = "target_forbidden" // 目标地址不被等级策略允许（如 guest 指向非回环）
	ErrBadRequest      = "bad_request"
)

// Register 是 wss 升级后 local 发送的第一条消息（JSON 文本帧）。
type Register struct {
	Type               string `json:"type"` // 固定 "register"
	Version            int    `json:"version"`
	ClientName         string `json:"client_name"`
	RequestedSubdomain string `json:"requested_subdomain"` // 空 = 由 hub 随机分配
	Target             string `json:"target,omitempty"`    // 归一化后的 host:port，hub 做等级策略校验

	// 短线重连认领（仅随机名隧道）：hub 断线保留期内携带上次的
	// 名字与凭证可拿回原域名；两者必须成对出现，任一为空即忽略。
	ResumeSubdomain string `json:"resume_subdomain,omitempty"`
	ResumeToken     string `json:"resume_token,omitempty"`
}

// Registered 是 hub 对 Register 的成功应答，此后连接进入 yamux 数据面。
type Registered struct {
	Type         string   `json:"type"` // 固定 "registered"
	TunnelID     string   `json:"tunnel_id"`
	Name         string   `json:"name"` // 实际分配的子域名（重连应答可能与请求不同，以本字段为准）
	URLs         []string `json:"urls"`
	Tier         string   `json:"tier"`
	TunnelTTLSec int64    `json:"tunnel_ttl_sec,omitempty"` // >0 表示隧道会被强制回收
	// ResumeToken 是下次断线重连认领本域名的凭证，每次注册轮换；
	// 仅随机名隧道下发（固定域名靠持久预留保护，无需此机制）。
	ResumeToken string `json:"resume_token,omitempty"`
}

// Error 是 hub 的失败应答。
type Error struct {
	Type    string `json:"type"` // 固定 "error"
	Code    string `json:"code"`
	Message string `json:"message"`
}
