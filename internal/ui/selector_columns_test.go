package ui

import (
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAcceptSelectorColumn(t *testing.T) {
	previousShell := config.Get().Shell
	config.Get().Shell = "/bin/sh"
	t.Cleanup(func() { config.Get().Shell = previousShell })
	for _, tc := range []struct{ name, args, single, multi string }{
		{"select column without map", "--delimiter : --select-column 1", "alpha", "alpha:beta"},
		{"select column with map", `--delimiter : --select-column 1 --map "printf mapped"`, "mapped", "mapped:mapped"},
		{"display column only", "--delimiter : --column 2", "alpha:First", "alpha:First:beta:Second"},
		{"no transform", "", "alpha:First", "alpha:First,beta:Second"},
	} {
		for _, multi := range []bool{false, true} {
			mode := "single"
			if multi {
				mode = "multi"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				m, index := setupTestModel()
				m.executor = executor.NewExecutor(index)
				v := &index.Cheats[0].Vars[0]
				v.Args = tc.args
				v.Shell = "printf 'alpha:First\\nbeta:Second\\n'"
				want := tc.single
				if multi {
					v.Args += " --multi"
					want = tc.multi
				}
				m.selected = index.Cheats[0]
				m.startVarResolutionInternal()
				cmd := m.prepareCurrentVar()
				if cmd == nil {
					t.Fatal("shell selector did not start")
				}
				m.Update(cmd())
				if multi {
					m.handleVarResolveKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
					m.handleVarResolveKey(tea.KeyMsg{Type: tea.KeyDown})
					m.handleVarResolveKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
				}
				m.handleVarResolveKey(tea.KeyMsg{Type: tea.KeyEnter})
				if got := m.selected.Scope["var"]; got != want {
					t.Fatalf("selected value = %q, want %q", got, want)
				}
			})
		}
	}
}
