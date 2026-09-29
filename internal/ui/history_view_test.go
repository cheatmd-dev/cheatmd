package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/internal/history"
)

func TestHistoryMissingCheatSearchesRecordedCommand(t *testing.T) {
	m, _ := setupTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.histState = &historyState{
		picker: NewPicker([]history.Entry{{
			File: "missing.md", Header: "Gone", Command: "unique absent command",
		}}, nil),
		prevInput:  "old search",
		prevCursor: 99,
		prevOffset: 99,
	}
	m.phase = phaseHistory

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := m.textInput.Value(); got != "unique absent command" {
		t.Fatalf("search = %q, want recorded command", got)
	}
	if got := m.textInput.Position(); got != len("unique absent command") {
		t.Fatalf("cursor = %d, want end of recorded command", got)
	}
	if m.picker.Cursor != 0 || m.picker.Offset != 0 {
		t.Fatal("picker cursor and offset did not reset")
	}
	if m.phase != phaseCheatSelect || m.histState != nil {
		t.Fatal("history did not close into cheat search")
	}
	if len(m.picker.Filtered) != 0 {
		t.Fatalf("got %d matches for absent command, want 0", len(m.picker.Filtered))
	}
	if view := StripANSI(m.View()); !strings.Contains(view, "unique absent command") {
		t.Fatalf("recorded command missing from search view: %q", view)
	}
}

func TestHistoryExistingCheatPrefillsRecordedScope(t *testing.T) {
	m, index := setupTestModel()
	cheat := index.Cheats[0]
	m.histState = &historyState{
		picker: NewPicker([]history.Entry{{
			File: cheat.File, Header: cheat.Header, Command: "echo recorded",
			Scope: map[string]string{"var": "recorded"},
		}}, nil),
		prevInput: "old search",
	}
	m.phase = phaseHistory

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.selected != cheat || m.phase != phaseVarResolve || m.histState != nil {
		t.Fatal("history did not open the existing cheat's variable prompt")
	}
	if got := m.textInput.Value(); got != "recorded" {
		t.Fatalf("variable prefill = %q, want recorded", got)
	}
	if got := cheat.Scope["var"]; got != "recorded" {
		t.Fatalf("scope var = %q, want recorded", got)
	}
}
