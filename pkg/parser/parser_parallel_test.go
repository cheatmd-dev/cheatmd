package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDirectoryParallelErrors(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test_cheats_err")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create 100 files with parse errors
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("# Cheat %d\n```bash\necho %d\n```\n<!-- cheat\nvar x = \n-->\n", i, i)
		err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file_%d.md", i)), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	p := NewParser()
	index, _ := p.ParseDirectory(dir)

	if len(index.Errors) != 100 {
		t.Errorf("expected 100 errors, got %d", len(index.Errors))
	}
}
