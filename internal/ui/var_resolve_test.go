package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gubarz/cheatmd/pkg/parser"
)

func TestVarResolveArrowKeysMoveSelection(t *testing.T) {
	m := newMainModel([]*parser.Cheat{{Header: "One"}}, parser.NewCheatIndex(), nil)
	items := []FilteredOption{
		{Display: "alpha", Original: "alpha", SearchText: "alpha"},
		{Display: "beta", Original: "beta", SearchText: "beta"},
		{Display: "gamma", Original: "gamma", SearchText: "gamma"},
	}
	m.phase = phaseVarResolve
	m.varState = &varResolveState{
		isPromptOnly: false,
		picker: NewPicker(items, func(opt FilteredOption, words []string) bool {
			return matchesAllWords(opt.SearchText, words)
		}),
	}

	model, _ := m.updateVarResolve(tea.KeyMsg{Type: tea.KeyDown})
	got := model.(*mainModel)
	if got.varState.picker.Cursor != 1 {
		t.Fatalf("cursor after down = %d, want 1", got.varState.picker.Cursor)
	}

	model, _ = got.updateVarResolve(tea.KeyMsg{Type: tea.KeyUp})
	got = model.(*mainModel)
	if got.varState.picker.Cursor != 0 {
		t.Fatalf("cursor after up = %d, want 0", got.varState.picker.Cursor)
	}
}

func TestVarResolvePrefillFiltersSelectionOptions(t *testing.T) {
	m := newMainModel([]*parser.Cheat{{Header: "One"}}, parser.NewCheatIndex(), nil)
	m.phase = phaseVarResolve
	m.varState = &varResolveState{
		vars: []varState{
			{
				def:     parser.VarDef{Name: "mode"},
				prefill: "publish",
			},
		},
	}

	model, _ := m.handleShellResult(shellResultMsg{
		options: []string{
			"preview\tPreview changes",
			"publish\tPublish changes",
			"archive\tArchive output",
		},
	})
	got := model.(*mainModel)

	if got.textInput.Value() != "publish" {
		t.Fatalf("input = %q, want publish", got.textInput.Value())
	}
	if got.varState.picker == nil {
		t.Fatal("picker is nil")
	}
	if len(got.varState.picker.Filtered) != 1 {
		t.Fatalf("filtered options = %d, want 1", len(got.varState.picker.Filtered))
	}
	if got.varState.picker.Filtered[0].Display != "publish\tPublish changes" {
		t.Fatalf("filtered option = %q", got.varState.picker.Filtered[0].Display)
	}
}

func TestVarResolveHandleShellResultPrefillMovesCursor(t *testing.T) {
	options := []string{"alpha", "beta", "gamma", "test.testing.test", "testing.test"}

	tests := []struct {
		name         string
		prefill      string
		wantCursor   int
		wantInput    string
		wantFiltered int
	}{
		{
			// "beta" filters the list to ["beta"]; exact match sits at Filtered[0].
			name:         "prefill matching second option lands at filtered index 0",
			prefill:      "beta",
			wantCursor:   0,
			wantInput:    "beta",
			wantFiltered: 1,
		},
		{
			// "alpha" filters to ["alpha"]; exact match at Filtered[0].
			name:         "prefill matching first option lands at filtered index 0",
			prefill:      "alpha",
			wantCursor:   0,
			wantInput:    "alpha",
			wantFiltered: 1,
		},
		{
			// "gamma" filters to ["gamma"]; exact match at Filtered[0].
			name:         "prefill matching last option lands at filtered index 0",
			prefill:      "gamma",
			wantCursor:   0,
			wantInput:    "gamma",
			wantFiltered: 1,
		},
		{
			// No prefill: Filter("") keeps all items, cursor stays at 0.
			name:         "no prefill leaves cursor at zero with all options visible",
			prefill:      "",
			wantCursor:   0,
			wantInput:    "",
			wantFiltered: 5,
		},
		{
			// "delta" matches nothing; Filtered is empty, cursor clamped to 0.
			name:         "prefill with no match leaves empty filtered list cursor at zero",
			prefill:      "delta",
			wantCursor:   0,
			wantInput:    "delta",
			wantFiltered: 0,
		},
		{
			// "testing.test" filters the list to ["test.testing.test", "testing.test"]; exact match sits at Filtered[1].
			name:         "prefill matching substring ",
			prefill:      "testing.test",
			wantCursor:   1,
			wantInput:    "testing.test",
			wantFiltered: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMainModel([]*parser.Cheat{{Header: "One"}}, parser.NewCheatIndex(), nil)
			m.phase = phaseVarResolve
			m.varState = &varResolveState{
				vars: []varState{
					{
						def:     parser.VarDef{Name: "TARGET"},
						prefill: tt.prefill,
					},
				},
			}

			model, _ := m.handleShellResult(shellResultMsg{options: options})
			got := model.(*mainModel)

			if got.varState.picker == nil {
				t.Fatal("picker is nil")
			}
			if len(got.varState.picker.Filtered) != tt.wantFiltered {
				t.Fatalf("len(Filtered) = %d, want %d", len(got.varState.picker.Filtered), tt.wantFiltered)
			}
			if got.varState.picker.Cursor != tt.wantCursor {
				t.Fatalf("cursor = %d, want %d", got.varState.picker.Cursor, tt.wantCursor)
			}
			if got.textInput.Value() != tt.wantInput {
				t.Fatalf("input = %q, want %q", got.textInput.Value(), tt.wantInput)
			}
		})
	}
}
