package main

import (
	"fmt"
	"os"
	"runtime"

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

// 版本信息由发布流程经 -ldflags 注入（version 必有，commit/date 在 GoReleaser 构建中注入）。
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// 环境变量覆盖层：优先级 flag > env > 配置文件 > 内置默认（见 resolveOne）。
const (
	envServer = "SOULINK_SERVER"
	envToken  = "SOULINK_TOKEN"
)

func main() {
	root := &cobra.Command{
		Use:     "soulink [target]",
		Short:   "Share local web services through a soulink hub",
		Version: version,
		// 运行期错误（如 Ctrl+C、认证失败）不属于用法错误，不打印 Usage；
		// 错误信息由 main 统一输出一次
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate(fmt.Sprintf("soulink %s (commit %s) %s/%s\nbuilt at: %s\nhttps://github.com/soulink/soulink-local\n",
		version, commit, runtime.GOOS, runtime.GOARCH, date))

	httpCmd, runHTTP := newHTTPCmd(root)
	// 裸目标等价 http 子命令：soulink 3000 ≡ soulink http 3000；无参数时打印帮助
	root.Args = cobra.MaximumNArgs(1)
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runHTTP(cmd, args)
	}

	root.AddCommand(httpCmd, newConfigCmd(), newCompletionCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// resolveOne 按 flag > env > 配置文件 > 默认值 解析单个配置项，返回生效值与来源标注。
func resolveOne(flagVal, envName, fileVal, defVal string) (value, source string) {
	if flagVal != "" {
		return flagVal, "flag"
	}
	if v := os.Getenv(envName); v != "" {
		return v, "env " + envName
	}
	if fileVal != "" {
		return fileVal, "file"
	}
	return defVal, "default"
}

// resolveDefaults 补齐运行参数（flag > env > 配置文件 > 内置默认）；
// token 的兜底是 guest 公开 token。配置文件损坏属显式错误，不得静默回落。
func resolveDefaults(opts *client.Options) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	opts.Server, _ = resolveOne(opts.Server, envServer, cfg.Server, defaultServer)
	opts.Token, _ = resolveOne(opts.Token, envToken, cfg.Token, "")
	if opts.Token == "" {
		opts.Token = defaultGuestToken
		fmt.Println("未配置 token，使用 guest 通道（随机域名，有时效限制）。执行 `soulink config set token <token>` 可获得固定域名。")
	}
	return nil
}
