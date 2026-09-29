package headless

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

type conditionalExecutor struct {
	*executor.Executor
	commands []string
}

func (e *conditionalExecutor) RunShell(command string) (string, error) {
	e.commands = append(e.commands, command)
	return "chosen", nil
}

func TestConditionalDefinitionsRespectSkippedValues(t *testing.T) {
	old := *config.Get()
	t.Cleanup(func() { *config.Get() = old })
	*config.Get() = config.DefaultConfig
	config.Get().AutoContinue = true
	t.Setenv("a", "")

	for _, tc := range []struct {
		name, literal, shell, prefill, condition, want string
		commands                                       []string
	}{
		{name: "false literal", literal: "should-be-skipped", condition: "$a == zeta"},
		{name: "false shell with prefill", shell: "printf 'should-be-skipped'", prefill: "cached", condition: "$a == zeta"},
		{name: "matching literal", literal: "chosen", condition: "$a == alpha", want: "chosen"},
		{name: "matching shell", shell: "printf 'chosen'", condition: "$a == alpha", want: "chosen", commands: []string{"printf 'chosen'"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("b", tc.prefill)
			cheat := &parser.Cheat{
				Header: "Conditional", Command: "echo [$b]",
				Vars: []parser.VarDef{
					{Name: "a"},
					{Name: "b", Literal: tc.literal, Shell: tc.shell, Condition: tc.condition},
				},
			}
			index := parser.NewCheatIndex()
			index.Cheats = []*parser.Cheat{cheat}
			exec := &conditionalExecutor{Executor: executor.NewExecutor(index)}
			var output bytes.Buffer
			session := &RunnerSession{
				Index: index, Exec: exec, Out: &output,
				Decoder: json.NewDecoder(strings.NewReader("{\"result\":{\"values\":{\"a\":\"alpha\"}}}\n{\"result\":{\"values\":{\"b\":\"chosen\"}}}\n")),
			}

			if err := session.Execute("Conditional", ""); err != nil {
				t.Fatal(err)
			}

			wantScope := map[string]string{"a": "alpha", "b": tc.want}
			if !reflect.DeepEqual(cheat.Scope, wantScope) {
				t.Errorf("scope = %v, want %v", cheat.Scope, wantScope)
			}
			if !reflect.DeepEqual(exec.commands, tc.commands) {
				t.Errorf("shell commands = %v, want %v", exec.commands, tc.commands)
			}
		})
	}
}
