package parser

import (
	"strings"
	"testing"
)

func TestANSIWarningMessage(t *testing.T) {
	p := NewParser()
	content := []byte("# \x1b[31mCheat\x1b[0m\n```bash\necho 1\n```\n")

	p.parseLines("test.md", content)

	// Create cheats so createCheat runs
	_ = p.index.Cheats

	if len(p.index.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(p.index.Errors))
	}

	errMessage := p.index.Errors[0].Message
	if !strings.Contains(errMessage, "Please remove them manually") {
		t.Errorf("expected warning to mention manual removal, got %q", errMessage)
	}
}
