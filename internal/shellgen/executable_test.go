package shellgen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWidgetQuotesExecutable(t *testing.T) {
	for _, tc := range []struct{ shell, want string }{
		{"bash", `'/tmp/tool\path'"'"'s name' --print`},
		{"zsh", `'/tmp/tool\path'"'"'s name' --print`},
		{"fish", `'/tmp/tool\\path\'s name' --print`},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			script, err := Widget(tc.shell, `/tmp/tool\path's name`)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(script, tc.want); got != 2 {
				t.Fatalf("script contains quoted executable %d times, want 2:\n%s", got, script)
			}
		})
	}
}

func TestWidgetInvokesQuotedExecutable(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is unavailable")
	}
	executable := filepath.Join(t.TempDir(), "example tool's name")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '<%s>\\n' \"$0\" \"$@\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	script, err := Widget("bash", executable)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"", "value with spaces; $(printf unwanted) 'quote'"} {
		command := script + "\nREADLINE_LINE=$1\n_cheatmd_widget || exit $?\nprintf '%s' \"$READLINE_LINE\"\n"
		out, err := exec.Command(bash, "--noprofile", "--norc", "-c", command, "widget-test", input).Output()
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("<%s>\n<--print>", executable)
		if input != "" {
			want += "\n<--match>\n<" + input + ">"
		}
		if string(out) != want {
			t.Fatalf("output = %q, want %q", out, want)
		}
	}
}
