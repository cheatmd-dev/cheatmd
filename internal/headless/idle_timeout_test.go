package headless

import (
	"testing"
	"time"
)

func TestCaptureIdleTimeout(t *testing.T) {
	previous := IdleTimeout
	IdleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { IdleTimeout = previous })

	for _, tc := range []struct {
		name, command, stdout string
	}{
		{"silent", "exec sleep 2", ""},
		{"output before idle", "printf ready; exec sleep 2", "ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCommandAndCapture("sh", tc.command)
			if err == nil {
				t.Fatal("expected the idle command to be terminated")
			}
			if stdout != tc.stdout || stderr != "" {
				t.Fatalf("captured stdout=%q stderr=%q, want stdout=%q and empty stderr", stdout, stderr, tc.stdout)
			}
		})
	}
}
