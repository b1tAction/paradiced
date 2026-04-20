// Package command provides CLI commands for paradiced CLI.
package command

import (
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() error {
	// Add interactive game commands
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(joinCmd)

	return rootCmd.Execute()
}

// rootCmd is the base command for pdcli.
var rootCmd = &cobra.Command{
	Use:   "pdcli",
	Short: "ParaDiced CLI - Verify game backend playability",
	Long: `ParaDiced CLI is a command-line tool for verifying ParaDiced game backend playability.

Features:
- Batch create/login test players
- Create and join paradiced_match
- Auto execute actions (roll_dice, user_choice)
- Game end report (success rate, duration, error stats)

Interactive Mode:
- pdcli play  - Connect to server
- pdcli host  - Create a room
- pdcli join  - Join a room`,
}

// AddCommand adds a subcommand to the root command.
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
