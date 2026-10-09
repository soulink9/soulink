package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/soulink/soulink-local/client"
)

// newHTTPCmd 构建 http 子命令，并返回其 RunE 供 root 的裸目标形式复用
// （root 与 http 共享同一组 flag，底层写入同一份 opts）。
func newHTTPCmd(root *cobra.Command) (*cobra.Command, func(cmd *cobra.Command, args []string) error) {
	var opts client.Options
	run := func(cmd *cobra.Command, args []string) error {
		opts.Target = args[0]
		if err := resolveDefaults(&opts); err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return client.Run(ctx, opts)
	}

	cmd := &cobra.Command{
		Use:   "http <target>",
		Short: "Expose a local HTTP/WebSocket service",
		Long: `Expose a local HTTP/WebSocket service through the hub.

<target> 支持以下形式：
  3000                      本机端口（等价 127.0.0.1:3000）
  localhost:3000            host:port
  http://192.168.1.10:8080  HTTP 目标（缺省端口 80）
  https://www.example.com   HTTPS 目标（缺省端口 443，TLS 拨号）

注意：guest 隧道只能指向 127.0.0.1 / localhost；注册用户可指向任意 IP 或域名。

参数优先级：--server/--token flag > SOULINK_SERVER/SOULINK_TOKEN 环境变量 >
~/.soulink/config.yaml（soulink config set 写入）> 内置默认。`,
		Args: cobra.ExactArgs(1),
		RunE: run,
	}
	cmd.Flags().StringVar(&opts.Server, "server", "", "hub address (default: env/config file, else built-in)")
	cmd.Flags().StringVar(&opts.Token, "token", "", "access token (default: env/config file, else guest)")
	cmd.Flags().StringVar(&opts.Subdomain, "subdomain", "", "requested subdomain (registered users only)")
	cmd.Flags().BoolVar(&opts.Insecure, "insecure", false, "skip TLS verification for wss")
	root.Flags().AddFlagSet(cmd.Flags())
	return cmd, run
}
