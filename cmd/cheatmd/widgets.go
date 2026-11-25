package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	script, err := shellgen.Widget(args[0], widgetExecutablePath(os.Args[0], executable))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), script)
	return err
}

// Prefer the path the user invoked when it identifies the running binary.
// A package manager can then repoint an installation symlink across upgrades.
func widgetExecutablePath(invoked, running string) string {
	path, err := exec.LookPath(invoked)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return running
	}
	if !filepath.IsAbs(path) {
		dir, err := os.Getwd()
		if err != nil {
			return running
		}
		path = dir + string(filepath.Separator) + path
	}
	invokedInfo, err := os.Stat(path)
	if err != nil {
		return running
	}
	runningInfo, err := os.Stat(running)
	if err != nil || !os.SameFile(invokedInfo, runningInfo) {
		return running
	}
	return path
}
