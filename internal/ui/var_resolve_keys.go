package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

// handleVarResolveKey processes keyboard input during variable resolution.
func (m *mainModel) handleVarResolveKey(msg tea.KeyMsg) tea.Cmd {
	if cmd := m.handleVarResolveNavKey(msg); cmd != nil {
		return cmd
	}

	switch msg.String() {
	case "enter":
		return m.acceptVarValue(false)
	case "alt+enter", "ctrl+j":
		return m.acceptVarValue(true)
	case "up", "ctrl+p", "down", "ctrl+n", "pgup", "pgdown":
		if m.varState.pathPicker != nil {
			m.varState.pathPicker.HandleKey(msg)
			if opt, ok := m.varState.pathPicker.Selected(); ok {
				runes := []rune(m.textInput.Value())
				newRunes := make([]rune, 0)
				newRunes = append(newRunes, runes[:m.varState.pathTokenStart]...)
				newRunes = append(newRunes, []rune(opt.Token)...)
				newCursor := len(newRunes)
				if m.varState.pathTokenEnd < len(runes) {
					newRunes = append(newRunes, runes[m.varState.pathTokenEnd:]...)
				}
				m.textInput.SetValue(string(newRunes))
				m.textInput.SetCursor(newCursor)
				m.varState.pathTokenEnd = newCursor
			}
			return func() tea.Msg { return nil } // bypass text input
		} else if !m.varState.isPromptOnly && m.varState.picker != nil {
			m.varState.picker.HandleKey(msg)
		}
		return nil
	case "tab":
		return m.handleVarResolveTab(msg, 1)
	case "shift+tab":
		return m.handleVarResolveTab(msg, -1)
	case " ":
		if cmd := m.handleVarResolveSpace(msg); cmd != nil {
			return cmd
		}
	default:
		return m.handleVarResolveDefaultKey(msg)
	}
	return nil
}

func (m *mainModel) handleVarResolveNavKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		m.selected = nil
		return tea.Quit
	case "esc":
		if m.varState.currentIdx > 0 {
			m.varState.currentIdx--
			vs := &m.varState.vars[m.varState.currentIdx]
			vs.resolved = false
			vs.value = ""
			vs.skipAutoCont = true
			m.textInput.SetValue("")
			if m.varState.picker != nil {
				m.picker.Cursor = 0
				m.picker.Offset = 0
			}
			return m.prepareCurrentVar()
		}
		m.phase = phaseCheatSelect
		m.varState = nil
		m.selected = nil
		m.textInput.SetValue(m.lastQuery)
		m.textInput.Placeholder = "Type to search..."
		m.filterCheats()
		m.picker.Cursor = 0
		m.picker.Offset = 0
		return nil
	}
	return nil
}

func (m *mainModel) handleVarResolveTab(msg tea.KeyMsg, dir int) tea.Cmd {
	if m.completePathFromInput(dir) {
		return func() tea.Msg { return nil } // bypass text input to preserve completion state
	}
	if !m.varState.isPromptOnly && m.varState.picker != nil {
		if opt, ok := m.varState.picker.Selected(); ok {
			m.textInput.SetValue(opt.Display)
			m.textInput.CursorEnd()
			return func() tea.Msg { return nil } // bypass text input
		}
	}
	return nil
}

func (m *mainModel) handleVarResolveSpace(msg tea.KeyMsg) tea.Cmd {
	if m.varState.selectOpts.Multi && !m.varState.isPromptOnly && m.varState.picker != nil {
		if opt, ok := m.varState.picker.Selected(); ok {
			vs := &m.varState.vars[m.varState.currentIdx]
			original := opt.Original
			if vs.multiSelectedSet[original] {
				vs.multiSelectedSet[original] = false
				for i, val := range vs.multiSelected {
					if val == original {
						vs.multiSelected = append(vs.multiSelected[:i], vs.multiSelected[i+1:]...)
						break
					}
				}
			} else {
				vs.multiSelectedSet[original] = true
				vs.multiSelected = append(vs.multiSelected, original)
			}
			return func() tea.Msg { return nil } // Return a non-nil dummy command to bypass text input
		}
	}
	return nil
}

func (m *mainModel) handleVarResolveDefaultKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == config.Get().KeyOpen {
		if m.varState != nil && m.varState.cheat != nil {
			openFileInViewer(m.varState.cheat.File)
		}
	}
	if msg.String() == config.Get().KeySubstitute {
		if m.enterSubstituteSearch() {
			return tea.Batch(tea.ClearScreen, textinput.Blink)
		}
	}
	if msg.String() == config.Get().KeyPreview {
		if m.varState != nil && m.varState.cheat != nil {
			if m.enterPreview(m.varState.cheat) {
				return tea.ClearScreen
			}
		}
	}
	return nil
}

func (m *mainModel) completePathFromInput(dir int) bool {
	if m.varState == nil {
		return false
	}

	// If path picker is active, this is a tab cycle request
	if m.varState.pathPicker != nil && len(m.varState.pathPicker.Filtered) > 0 {
		p := m.varState.pathPicker
		if dir > 0 {
			if p.Cursor >= len(p.Filtered)-1 {
				p.Cursor = 0
				p.Offset = 0
			} else {
				p.MoveCursor(1)
			}
		} else {
			if p.Cursor <= 0 {
				p.Cursor = len(p.Filtered) - 1
				p.Offset = max(0, p.Cursor-9)
			} else {
				p.MoveCursor(-1)
			}
		}

		if opt, ok := p.Selected(); ok {
			runes := []rune(m.textInput.Value())
			newRunes := make([]rune, 0)
			newRunes = append(newRunes, runes[:m.varState.pathTokenStart]...)
			newRunes = append(newRunes, []rune(opt.Token)...)
			newCursor := len(newRunes)
			if m.varState.pathTokenEnd < len(runes) {
				newRunes = append(newRunes, runes[m.varState.pathTokenEnd:]...)
			}

			m.textInput.SetValue(string(newRunes))
			m.textInput.SetCursor(newCursor)
			m.varState.pathTokenEnd = newCursor
		}
		return true
	}

	value := m.textInput.Value()
	cursor := m.textInput.Position()
	if !m.varState.isPromptOnly && !looksPathLikeInput(value, cursor) {
		return false
	}
	result, ok := completePathValue(value, cursor)
	if !ok {
		m.clearPathCompletions()
		return false
	}

	m.textInput.SetValue(result.Value)
	m.textInput.SetCursor(result.Cursor)

	if len(result.Candidates) > 1 {
		m.varState.pathPicker = NewPicker(result.Candidates, func(item pathCompletionCandidate, words []string) bool {
			return true
		})
		runes := []rune(result.Value)
		m.varState.pathTokenStart = pathTokenStart(runes, result.Cursor)
		m.varState.pathTokenEnd = result.Cursor
	} else {
		m.varState.pathPicker = nil
	}

	if !m.varState.isPromptOnly && m.varState.picker != nil {
		m.varState.picker.Filter(m.textInput.Value())
	}
	return true
}

func looksPathLikeInput(value string, cursor int) bool {
	runes := []rune(value)
	if cursor < 0 || cursor > len(runes) {
		cursor = len(runes)
	}
	start := pathTokenStart(runes, cursor)
	token := strings.TrimLeft(string(runes[start:cursor]), `"'`)
	return strings.HasPrefix(token, "/") ||
		strings.HasPrefix(token, "./") ||
		strings.HasPrefix(token, "../") ||
		strings.HasPrefix(token, "~/") ||
		strings.HasPrefix(token, "$") ||
		strings.Contains(token, "/")
}

func (m *mainModel) clearPathCompletions() {
	if m.varState == nil {
		return
	}
	m.varState.pathPicker = nil
}

// acceptVarValue accepts the current value and moves to next variable.
func (m *mainModel) acceptVarValue(bypassSelection bool) tea.Cmd {
	if m.varState == nil {
		return tea.Quit
	}

	vs := &m.varState.vars[m.varState.currentIdx]
	var value string

	if bypassSelection || m.varState.isPromptOnly {
		value = m.textInput.Value()
	} else if m.varState.selectOpts.Multi && len(vs.multiSelected) > 0 {
		var mapped []string
		for _, selected := range vs.multiSelected {
			if m.varState.selectOpts.MapCmd != "" {
				selected = applyMapTransform(selected, m.varState.selectOpts)
			}
			mapped = append(mapped, selected)
		}
		delim := m.varState.selectOpts.Delimiter
		if delim == "" {
			delim = ","
		}
		value = strings.Join(mapped, delim)
	} else if m.varState.picker != nil {
		if opt, ok := m.varState.picker.Selected(); ok {
			selected := opt.Original
			if m.varState.selectOpts.MapCmd != "" {
				selected = applyMapTransform(selected, m.varState.selectOpts)
			}
			value = selected
		} else {
			value = m.textInput.Value()
		}
	} else {
		value = m.textInput.Value()
	}

	vs.value = value
	vs.resolved = true
	m.varState.currentIdx++

	m.textInput.SetValue("")
	m.clearPathCompletions()
	m.picker.Cursor = 0
	m.picker.Offset = 0

	return m.prepareCurrentVar()
}
