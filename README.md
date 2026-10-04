# soulink-local

soulink 客户端：一条命令把本地 Web 服务分享出去。

## 用法

```bash
# 随机子域名（guest token）
soulink http 3000 --server localhost:18080 --token dev-guest-token

# 指定子域名
soulink http 3000 --server localhost:18080 --token dev-guest-token --subdomain demo

# 局域网其他机器
soulink http 192.168.1.10:8080 --server hub.example.com
```

`--server` 不带 scheme 时默认 `ws://`；生产环境使用 `wss://hub.example.com`。
`--insecure` 可跳过自签证书校验。

## proto/

`proto/` 是 hub 与 local 共享的协议契约包（握手 DTO + wsconn 适配器），
hub 以 Go module 方式引用本仓库。
