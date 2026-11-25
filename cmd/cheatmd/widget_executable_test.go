package main

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestWidgetExecutablePathKeepsInstallationSymlink(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "version-1")
	second := filepath.Join(root, "version-2")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	installName := "cheatmd-test"
	installPath := filepath.Join(root, installName)
	if err := os.Symlink(first, installPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("PATH", root)

	for _, invoked := range []string{installName, installPath} {
		if got := widgetExecutablePath(invoked, first); got != installPath {
			t.Errorf("widgetExecutablePath(%q) = %q, want %q", invoked, got, installPath)
		}
	}
	if err := os.Remove(installPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, installPath); err != nil {
		t.Fatal(err)
	}
	if got := widgetExecutablePath(installName, first); got != first {
		t.Errorf("different file returned %q, want running executable %q", got, first)
	}
}
