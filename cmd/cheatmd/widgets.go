package main

import (
	"fmt"

	"github.com/cheatmd-dev/cheatmd/internal/shellgen"
	"github.com/spf13/cobra"
)

var widgetCmd = &cobra.Command{
	Use:   "widget [shell]",
	Short: "Output shell widget script for integration",
	Long: `Outputs a shell script that can be sourced for shell integration.

Usage:
  eval "$(cheatmd widget bash)"

Then press Ctrl+G to trigger the cheatmd selector.`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE:      runWidget,
}

func runWidget(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		fmt.Fprint(cmd.OutOrStdout(), shellgen.BashWidget())
	case "zsh":
		fmt.Fprint(cmd.OutOrStdout(), shellgen.ZshWidget())
	case "fish":
		fmt.Fprint(cmd.OutOrStdout(), shellgen.FishWidget())
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
	return nil
}
