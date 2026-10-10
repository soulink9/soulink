# soulink

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

soulink 客户端：一条命令把本地 Web 服务（含 WebSocket）分享出去。
单二进制、无依赖，支持 linux / darwin / windows（amd64、arm64）。

```bash
go install github.com/soulink9/soulink/cmd/soulink@latest
```

## 安装

从 Releases 下载对应平台归档，或自行构建：

```bash
make build   # 本机平台 → bin/soulink
make dist    # 六平台归档 + checksums → dist/
```

## 用法

```bash
# 1. 配置（在 hub 网站注册并创建 token 后写入）
soulink config set server https://hub.soulink.cam   # 官方部署可省略（内置默认）
soulink config set token slk_xxx                    # 自托管 hub 时先 set server

soulink config show    # 查看生效配置（token 打码，标注来源）
soulink config unset token

# 2. 分享本地 3000 端口，指定子域名
soulink http 3000 --subdomain demo
# → https://demo.hub.example.com

# 免配置（guest 通道：随机域名、限时回收）
soulink http 3000

# 分享局域网内其他机器
soulink http 192.168.1.10:8080
```

参数优先级：`--server` / `--token` 命令行 > `SOULINK_SERVER` / `SOULINK_TOKEN` 环境变量
（适合 CI/容器）> `~/.soulink/config.yaml`（config set 写入）> 内置默认
（server 缺省为 `https://hub.soulink.cam`，token 缺省走 guest 通道）。
`--insecure` 可跳过 hub 自签证书校验。

shell 自动补全：`soulink completion bash|zsh|fish|powershell`（按命令输出的说明安装）。

> 内置默认地址是「官方部署」身份的一部分，与内置 guest token 成对。覆盖方式：
> 手工构建用 `-ldflags "-X main.defaultServer=https://your-hub"`（**置空**即恢复「每次必须显式指定 `--server`」）；
> 经 Makefile 出包则设环境变量 `SOULINK_DEFAULT_SERVER=https://your-hub`，不设即沿用源码内置值，无需改发布配置。

断线自动重连（指数退避），重连后尽量保持原子域名。

## proto/

`proto/` 是 hub 与 local 共享的协议契约包（握手 DTO + wsconn 适配器），
hub 以 Go module 方式引用本仓库。
