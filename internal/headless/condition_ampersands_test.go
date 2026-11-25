package headless

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestHeadlessConditionWithLiteralAmpersands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "condition.md")
	const cheat = "# Flavor\n\n" +
		"```sh\nprintf '%s' $result\n```\n" +
		"<!-- cheat\n" +
		"var flavor\n" +
		"var result := fallback\n" +
		"if $flavor == Rock && Roll\n" +
		"  if enabled\n" +
		"    var result := matched\n" +
		"  fi\n" +
		"fi\n" +
		"-->\n"
	if err := os.WriteFile(path, []byte(cheat), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ flavor, want string }{
		{"Rock && Roll", "matched"},
		{"Rock", "fallback"},
	} {
		t.Run(tc.flavor, func(t *testing.T) {
			index, err := parser.NewParser().ParseSingleFile(path)
			if err != nil {
				t.Fatal(err)
			}
			input, err := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"result":  map[string]any{"values": map[string]string{"flavor": tc.flavor}},
				"id":      1,
			})
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			session := &RunnerSession{
				Cheat: index.Cheats[0], Index: index, Exec: executor.NewExecutor(index),
				Decoder: json.NewDecoder(bytes.NewReader(input)), Out: &output,
			}
			session.initializeVariables()
			if err := session.resolveInteractively(); err != nil {
				t.Fatal(err)
			}
			if got := session.Exec.BuildFinalCommand(session.Cheat); got != "printf '%s' "+tc.want {
				t.Fatalf("command = %q, want result %q", got, tc.want)
			}
			if !strings.Contains(output.String(), `"name":"flavor"`) {
				t.Fatalf("missing flavor prompt: %s", &output)
			}
		})
	}
}
