package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestHeadlessStdoutRedirect(t *testing.T) {
	// Replicate the behavior in runCheats
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w

	// Simulate headless run
	headlessOut := os.Stdout
	os.Stdout = os.Stderr // the fix

	// Simulate a random print from the app
	os.Stdout.WriteString("warning: duplicate export\n")

	// Simulate headless writing json
	if _, err := headlessOut.WriteString(`{"jsonrpc":"2.0"}` + "\n"); err != nil {
		t.Fatalf("failed to write headless output: %v", err)
	}
	w.Close()

	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	output := buf.String()

	if strings.Contains(output, "warning") {
		t.Errorf("expected warning to be redirected, got %s", output)
	}
	if !strings.Contains(output, "jsonrpc") {
		t.Errorf("expected jsonrpc in stdout, got %s", output)
	}
}
