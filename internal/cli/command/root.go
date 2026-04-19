// Package command provides CLI commands for paradiced CLI.
package command

import (
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// rootCmd is the base command for pdcli.
var rootCmd = &cobra.Command{
	Use:   "pdcli",
	Short: "ParaDiced CLI - 验证游戏后端可玩性",
	Long: `ParaDiced CLI 是用于快速验证 ParaDiced 游戏后端可玩性的命令行工具。

支持功能:
- 批量创建/登录测试玩家
- 创建与加入 paradiced_match
- 自动执行动作 (roll_dice, user_choice)
- 对局结束报告 (成功率、耗时、错误统计)`,
}

// AddCommand adds a subcommand to the root command.
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
