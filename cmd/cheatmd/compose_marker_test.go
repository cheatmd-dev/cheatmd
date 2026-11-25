package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestComposedCommandWithoutVariablesIsExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "composed.md")
	content := buildComposeMarkdown("Greeting", "", "echo hello")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	index, err := parser.NewParser().ParseSingleFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cheats := index.FilterByConfig(true)
	if len(cheats) != 1 {
		t.Fatalf("got %d executable cheats, want one", len(cheats))
	}
	if cheats[0].Command != "echo hello" {
		t.Fatalf("got command %q, want echo hello", cheats[0].Command)
	}
}
