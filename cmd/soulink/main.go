package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/soulink/soulink-local/client"
	"github.com/soulink/soulink-local/internal/config"
)

const defaultGuestToken = "soulink-guest" // 与 hub 默认配置一致的公开 token

// version 由发布流程经 -ldflags "-X main.version=..." 注入。
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "soulink",
		Short:   "Share local web services through a soulink hub",
		Version: version,
	}

	var opts client.Options
	httpCmd := &cobra.Command{
		Use:   "http <target>",
		Short: "Expose a local HTTP/WebSocket service",
		Long: `Expose a local HTTP/WebSocket service through the hub.

<target> 支持以下形式：
  3000                      本机端口（等价 127.0.0.1:3000）
  localhost:3000            host:port
  http://192.168.1.10:8080  HTTP 目标（缺省端口 80）
  https://www.example.com   HTTPS 目标（缺省端口 443，TLS 拨号）

注意：guest 隧道只能指向 127.0.0.1 / localhost；注册用户可指向任意 IP 或域名。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Target = args[0]
			resolveDefaults(&opts)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return client.Run(ctx, opts)
		},
	}
	httpCmd.Flags().StringVar(&opts.Server, "server", "", "hub address (default: from `soulink login`)")
	httpCmd.Flags().StringVar(&opts.Token, "token", "", "access token (default: from `soulink login`, else guest)")
	httpCmd.Flags().StringVar(&opts.Subdomain, "subdomain", "", "requested subdomain (registered users only)")
	httpCmd.Flags().BoolVar(&opts.Insecure, "insecure", false, "skip TLS verification for wss")

	var loginServer string
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Save hub address and your token locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if loginServer == "" {
				return fmt.Errorf("--server is required")
			}
			fmt.Printf("在 %s 登录后，进入「我的 Token」页面创建并复制 token。\n", loginServer)
			fmt.Print("粘贴 token: ")
			token, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return err
			}
			token = strings.TrimSpace(token)
			if token == "" {
				return fmt.Errorf("token is empty")
			}
			cfg, _ := config.Load()
			cfg.Server = loginServer
			cfg.Token = token
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("已保存到 ~/.soulink/config.yaml")
			return nil
		},
	}
	loginCmd.Flags().StringVar(&loginServer, "server", "", "hub address (e.g. https://hub.example.com)")

	root.AddCommand(httpCmd, loginCmd)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// resolveDefaults 按 flag > 配置文件 > guest 的顺序补齐 server/token。
func resolveDefaults(opts *client.Options) {
	cfg, _ := config.Load()
	if opts.Server == "" {
		opts.Server = cfg.Server
	}
	if opts.Token == "" {
		opts.Token = cfg.Token
	}
	if opts.Token == "" {
		opts.Token = defaultGuestToken
		fmt.Println("未登录，使用 guest 通道（随机域名，有时效限制）。执行 `soulink login` 可获得固定域名。")
	}
}
