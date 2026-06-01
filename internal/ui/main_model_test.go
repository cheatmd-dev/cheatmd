package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/internal/history"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

type mockExecutor struct{}

func (m *mockExecutor) RunShell(command string) (string, error)                       { return "", nil }
func (m *mockExecutor) BuildFinalCommand(cheat *parser.Cheat) string                  { return cheat.Command }
func (m *mockExecutor) OutputWithMode(command string, mode executor.OutputMode) error { return nil }

func setupTestModel() (*mainModel, *parser.CheatIndex) {
	cheat := &parser.Cheat{
		File:    "/test/path/cheat.md",
		Header:  "Test Cheat",
		Command: "echo $var",
		Vars: []parser.VarDef{
			{Name: "var"},
		},
	}
	index := parser.NewCheatIndex()
	index.Cheats = []*parser.Cheat{cheat}
	index.Root = "/test/path"

	m := newMainModel(index.Cheats, index, &mockExecutor{})
	return &m, index
}

func TestUIStability_NoHangs(t *testing.T) {
	m, _ := setupTestModel()

	// Initial phase should be cheat select
	if m.phase != phaseCheatSelect {
		t.Fatalf("expected phaseCheatSelect, got %v", m.phase)
	}

	// Send an unmapped key, should not quit or hang
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("unknown")})
	if m.quitting {
		t.Fatal("UI quit unexpectedly on unknown input")
	}

	// Send ctrl+c, should quit cleanly
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.quitting {
		t.Fatal("UI did not quit on ctrl+c")
	}
}

func TestUIHistoryMenu_EscapeReturnsToCheatSelect(t *testing.T) {
	m, _ := setupTestModel()

	// For test purposes, we mock entering history by forcefully setting the phase and state
	// (since history file might not exist in tests)
	m.histState = &historyState{
		picker:     NewPicker[history.Entry](nil, nil),
		prevInput:  "search",
		prevCursor: 0,
		prevOffset: 0,
	}
	m.phase = phaseHistory

	// Send Esc key
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should safely return to cheat select
	if m.phase != phaseCheatSelect {
		t.Errorf("Expected to return to phaseCheatSelect, got %v", m.phase)
	}
	if m.histState != nil {
		t.Errorf("Expected histState to be nil after exit")
	}
}

func TestUIWidgetMatch_EscapeReturnsToCheatSelect(t *testing.T) {
	m, index := setupTestModel()

	// Manually trigger the transition as if handleMatchCmd found a perfect hit
	m.selected = index.Cheats[0]
	m.startVarResolutionInternal()

	if m.phase != phaseVarResolve {
		t.Fatalf("Expected phaseVarResolve, got %v", m.phase)
	}

	// Currently at the first variable prompt. Hit ESC.
	// This simulates the user cancelling the widget prompt.
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// It should gracefully fall back to the cheat list instead of crashing or quitting
	if m.phase != phaseCheatSelect {
		t.Errorf("Expected ESC to fall back to phaseCheatSelect, got %v", m.phase)
	}
	if m.selected != nil {
		t.Errorf("Expected selected cheat to be nil after aborting")
	}
	if m.varState != nil {
		t.Errorf("Expected varState to be nil after aborting")
	}
}
