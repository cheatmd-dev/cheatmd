package ui

import (
	"fmt"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

type shellRequestExecutor struct{ mockExecutor }

func (*shellRequestExecutor) RunShell(command string) (string, error) {
	if command == "failed" {
		return "", fmt.Errorf("stale shell failure")
	}
	return command + "\n" + command + "-other", nil
}

func TestShellResultsFollowCurrentPrompt(t *testing.T) {
	oldSyntax := config.Get().VarSyntax
	t.Cleanup(func() { config.Get().VarSyntax = oldSyntax })
	config.Get().VarSyntax = "dollar"
	t.Setenv("first", "")
	t.Setenv("second", "")

	for _, revisit := range []bool{false, true} {
		for _, firstValue := range []string{"old", "failed"} {
			t.Run(fmt.Sprintf("revisit=%t/result=%s", revisit, firstValue), func(t *testing.T) {
				m, index := setupTestModel()
				cheat := index.Cheats[0]
				cheat.Command = "echo $first $second"
				cheat.Vars = []parser.VarDef{{Name: "first"}, {Name: "second", Shell: "$first"}}
				m.executor = &shellRequestExecutor{}
				m.selected = cheat
				m.startVarResolution()
				m.textInput.SetValue(firstValue)
				_, delayed := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if delayed == nil {
					t.Fatal("shell variable returned no command")
				}

				m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				m.textInput.SetValue("editing")
				wantInput := "editing"
				wantIndex := 0
				var wantOptions []string
				if revisit {
					m.textInput.SetValue("current")
					_, current := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
					if current == nil {
						t.Fatal("revisited shell variable returned no command")
					}
					m.Update(current())
					wantOptions = []string{"current", "current-other"}
					wantInput = ""
					wantIndex = 1
					if !reflect.DeepEqual(m.varState.options, wantOptions) {
						t.Fatalf("current result options = %v, want %v", m.varState.options, wantOptions)
					}
				}

				m.Update(delayed())

				if m.varState.currentIdx != wantIndex || m.textInput.Value() != wantInput {
					t.Errorf("prompt after stale result = (%d, %q), want (%d, %q)", m.varState.currentIdx, m.textInput.Value(), wantIndex, wantInput)
				}
				if !reflect.DeepEqual(m.varState.options, wantOptions) {
					t.Errorf("options after stale result = %v, want %v", m.varState.options, wantOptions)
				}
				if m.varState.shellErr != nil {
					t.Errorf("stale error reached current prompt: %v", m.varState.shellErr)
				}
				if m.varState.isPromptOnly == revisit {
					t.Errorf("prompt-only mode = %t, want %t", m.varState.isPromptOnly, !revisit)
				}
			})
		}
	}
}
