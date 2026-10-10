package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soulink9/soulink/internal/config"
)

// newConfigCmd 构建 config 子命令族：path / show / set / unset。
// 这是配置的唯一写入入口（login 命令已被 config set 取代）。
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage local configuration (~/.soulink/config.yaml)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Println(p)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show effective config (token masked, with value sources)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key (" + strings.Join(config.ValidKeys, " | ") + ")",
		Long: `Set a config key in ~/.soulink/config.yaml.

可用键：
  server  hub 地址（如 https://hub.soulink.cam）
  token   访问令牌（在 hub 网站「我的 Token」页面创建）

token 值打码回显；文件权限恒为 0600。
也可用 SOULINK_SERVER / SOULINK_TOKEN 环境变量临时覆盖（优先级高于配置文件）。`,
		Example: `  soulink config set server https://hub.soulink.cam
  soulink config set token slk_xxx`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("value 不能为空")
			}
			if err := config.Set(key, value); err != nil {
				return err
			}
			p, _ := config.Path()
			shown := value
			if key == "token" {
				shown = maskToken(value)
			}
			fmt.Printf("已保存 %s: %s → %s\n", key, shown, p)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config key (" + strings.Join(config.ValidKeys, " | ") + ")",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Unset(args[0]); err != nil {
				return err
			}
			fmt.Printf("已移除 %s\n", args[0])
			return nil
		},
	})

	return cmd
}

// showConfig 打印配置文件路径与各项的生效值及来源（flag 不参与：show 自身无 --server/--token）。
func showConfig() error {
	p, err := config.Path()
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fmt.Printf("配置文件：%s\n", p)

	server, serverSrc := resolveOne("", envServer, cfg.Server, defaultServer)
	fmt.Printf("  server: %s  (%s)\n", server, serverSrc)

	token, tokenSrc := resolveOne("", envToken, cfg.Token, "")
	if token == "" {
		fmt.Println("  token:  (未设置，运行隧道时走 guest 通道)")
	} else {
		fmt.Printf("  token:  %s  (%s)\n", maskToken(token), tokenSrc)
	}
	return nil
}

// maskToken 打码 token：保留前 4 后 4，过短则整体打码。
func maskToken(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "…" + s[len(s)-4:]
}
