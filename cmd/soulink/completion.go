package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newCompletionCmd 生成各 shell 的自动补全脚本（cobra 内建生成器）。
// 自定义实现而非 cobra 默认命令，以统一 Short/中文说明风格。
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `生成 shell 自动补全脚本。

bash:       soulink completion bash > /etc/bash_completion.d/soulink
zsh:        soulink completion zsh > "${fpath[1]}/_soulink"
fish:       soulink completion fish > ~/.config/fish/completions/soulink.fish
powershell: soulink completion powershell | Out-String | Invoke-Expression`,
		Args:                  cobra.ExactArgs(1),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				err = cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				err = cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				err = fmt.Errorf("不支持的 shell %q（可用：bash, zsh, fish, powershell）", args[0])
			}
			return err
		},
	}
}
