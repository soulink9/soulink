# soulink-local

soulink 客户端：一条命令把本地 Web 服务（含 WebSocket）分享出去。
单二进制、无依赖，支持 linux / darwin / windows（amd64、arm64）。

## 安装

从 Releases 下载对应平台二进制，或自行构建：

```bash
go install github.com/soulink/soulink-local/cmd/soulink@latest
```

## 用法

```bash
# 1. 登录（在 hub 网站注册并创建 token 后粘贴）
soulink login --server https://hub.example.com

# 2. 分享本地 3000 端口，指定子域名
soulink http 3000 --subdomain demo
# → https://demo.hub.example.com
# → https://hub.example.com/s/demo/   （路径模式备用地址）

# 免登录（guest 通道：随机域名、限时回收）
soulink http 3000 --server hub.example.com

# 分享局域网内其他机器
soulink http 192.168.1.10:8080
```

参数优先级：`--server` / `--token` 命令行 > `~/.soulink/config.yaml`（login 写入）> guest 通道。
`--insecure` 可跳过 hub 自签证书校验。

断线自动重连（指数退避），重连后尽量保持原子域名。

## proto/

`proto/` 是 hub 与 local 共享的协议契约包（握手 DTO + wsconn 适配器），
hub 以 Go module 方式引用本仓库。
