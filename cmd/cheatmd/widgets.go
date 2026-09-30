package main

import (
	"fmt"
	"os"

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
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	script, err := shellgen.Widget(args[0], executable)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), script)
	return nil
}
