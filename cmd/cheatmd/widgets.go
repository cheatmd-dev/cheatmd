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
	var script string

	switch shell {
	case "bash":
		script = shellgen.BashWidget()
	case "zsh":
		script = shellgen.ZshWidget()
	case "fish":
		script = shellgen.FishWidget()
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), script)
	return err
}
