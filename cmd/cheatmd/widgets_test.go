package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type failingWidgetWriter struct {
	err error
}

func (w failingWidgetWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestWidgetOutput(t *testing.T) {
	for _, tc := range []struct {
		shell string
		want  string
	}{
		{"bash", `READLINE_LINE="$output"`},
		{"zsh", `BUFFER="$output"`},
		{"fish", `commandline -r "$output"`},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			t.Run("write error", func(t *testing.T) {
				wantErr := errors.New("widget output failed")
				cmd := &cobra.Command{}
				cmd.SetOut(failingWidgetWriter{err: wantErr})
				if err := runWidget(cmd, []string{tc.shell}); !errors.Is(err, wantErr) {
					t.Fatalf("runWidget error = %v, want %v", err, wantErr)
				}
			})
			t.Run("success", func(t *testing.T) {
				cmd := &cobra.Command{}
				var out bytes.Buffer
				cmd.SetOut(&out)
				if err := runWidget(cmd, []string{tc.shell}); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), tc.want) {
					t.Fatalf("output %q does not contain %q", out.String(), tc.want)
				}
			})
		})
	}
}
