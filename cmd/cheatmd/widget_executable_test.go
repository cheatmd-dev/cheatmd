package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWidgetUsesCurrentExecutable(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			cmd := &cobra.Command{}
			var out bytes.Buffer
			cmd.SetOut(&out)
			if err := runWidget(cmd, []string{shell}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), executable) {
				t.Fatalf("widget does not invoke current executable %q", executable)
			}
		})
	}
}

func TestWidgetRejectsUnsupportedShell(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := runWidget(cmd, []string{"elvish"})
	const want = "unsupported shell: elvish (supported: bash, zsh, fish)"
	if err == nil || err.Error() != want {
		t.Fatalf("runWidget error = %v, want %q", err, want)
	}
	if out.Len() != 0 {
		t.Fatalf("unsupported shell produced output: %q", out.String())
	}
}
