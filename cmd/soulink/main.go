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

// defaultServer 与 defaultGuestToken 共同描述「官方部署」的身份：hub 地址与 guest 公开 token
// 必须成对匹配（hub 侧 guest_token 不一致时 guest 通道回 401），换部署时两者一起改。
// 用 var 而非 const，以便经 -ldflags "-X main.defaultServer=..." 覆盖；置空即恢复「必须显式指定地址」。
// 必须带 scheme：client 对裸 host 补的是 ws://（明文），只有 https:// 才走 wss。
var defaultServer = "https://hub.soulink.cam"

// version 由发布流程经 -ldflags "-X main.version=..." 注入。
var version = "dev"

func main() {
	var opts client.Options
	runHTTP := func(cmd *cobra.Command, args []string) error {
		opts.Target = args[0]
		resolveDefaults(&opts)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return client.Run(ctx, opts)
	}

	root := &cobra.Command{
		Use:     "soulink [target]",
		Short:   "Share local web services through a soulink hub",
		Version: version,
		// 运行期错误（如 Ctrl+C、认证失败）不属于用法错误，不打印 Usage；
		// 错误信息由 main 统一输出一次
		SilenceUsage:  true,
		SilenceErrors: true,
		// 裸目标等价 http 子命令：soulink 3000 ≡ soulink http 3000；
		// 无参数时打印帮助
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runHTTP(cmd, args)
		},
	}

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
		RunE: runHTTP,
	}
	httpCmd.Flags().StringVar(&opts.Server, "server", "", "hub address (default: from soulink login, else built-in)")
	httpCmd.Flags().StringVar(&opts.Token, "token", "", "access token (default: from soulink login, else guest)")
	httpCmd.Flags().StringVar(&opts.Subdomain, "subdomain", "", "requested subdomain (registered users only)")
	httpCmd.Flags().BoolVar(&opts.Insecure, "insecure", false, "skip TLS verification for wss")
	// root 与 http 共享同一组 flag（底层都写入 opts）
	root.Flags().AddFlagSet(httpCmd.Flags())

	var loginServer string
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Save hub address and your token locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 未指定时用内置默认（默认被清空的构建下仍要求显式给出）
			server := loginServer
			if server == "" {
				server = defaultServer
			}
			if server == "" {
				return fmt.Errorf("--server is required")
			}
			fmt.Printf("在 %s 登录后，进入「我的 Token」页面创建并复制 token。\n", server)
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
			// 显式写入解析结果：配置文件记录的是用户的选择，不随后续二进制默认值漂移
			cfg.Server = server
			cfg.Token = token
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("已保存到 ~/.soulink/config.yaml")
			return nil
		},
	}
	loginCmd.Flags().StringVar(&loginServer, "server", "", "hub address (default: built-in)")

	root.AddCommand(httpCmd, loginCmd)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// resolveDefaults 补齐运行参数：server 为 flag > 配置文件 > 内置默认，token 为 flag > 配置文件 > guest。
func resolveDefaults(opts *client.Options) {
	cfg, _ := config.Load()
	if opts.Server == "" {
		opts.Server = cfg.Server
	}
	if opts.Server == "" {
		opts.Server = defaultServer
	}
	if opts.Token == "" {
		opts.Token = cfg.Token
	}
	if opts.Token == "" {
		opts.Token = defaultGuestToken
		fmt.Println("未登录，使用 guest 通道（随机域名，有时效限制）。执行 `soulink login` 可获得固定域名。")
	}
}
