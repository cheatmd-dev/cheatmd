package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

// handleVarResolveKey processes keyboard input during variable resolution.
func (m *mainModel) handleVarResolveKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		m.selected = nil
		return tea.Quit
	case "esc":
		// Go back to previous var or cheat selection.
		if m.varState.currentIdx > 0 {
			m.varState.currentIdx--
			vs := &m.varState.vars[m.varState.currentIdx]
			vs.resolved = false
			vs.value = ""
			vs.skipAutoCont = true
			m.textInput.SetValue("")
			m.picker.Cursor = 0
			m.picker.Offset = 0
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
	case "enter":
		return m.acceptVarValue(false)
	case "alt+enter", "ctrl+j":
		return m.acceptVarValue(true)
	case "up", "ctrl+p", "down", "ctrl+n", "pgup", "pgdown":
		if !m.varState.isPromptOnly && m.varState.picker != nil {
			m.varState.picker.HandleKey(msg)
		}
		return nil
	case "tab":
		if m.completePathFromInput() {
			return nil
		}
		if !m.varState.isPromptOnly && m.varState.picker != nil {
			if opt, ok := m.varState.picker.Selected(); ok {
				m.textInput.SetValue(opt.Display)
				m.textInput.CursorEnd()
			}
		}
	case " ":
		if m.varState.selectOpts.Multi && !m.varState.isPromptOnly && m.varState.picker != nil {
			if opt, ok := m.varState.picker.Selected(); ok {
				vs := &m.varState.vars[m.varState.currentIdx]
				original := opt.Original
				if vs.multiSelectedSet[original] {
					vs.multiSelectedSet[original] = false
					// Remove from multiSelected list
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
				// Don't pass space to text input, just return nil to re-render
				return nil
			}
		}
	default:
		if msg.String() == config.GetKeyOpen() {
			if m.varState != nil && m.varState.cheat != nil {
				openFileInViewer(m.varState.cheat.File)
			}
		}
		if msg.String() == config.GetKeySubstitute() {
			if m.enterSubstituteSearch() {
				return tea.Batch(tea.ClearScreen, textinput.Blink)
			}
		}
		if msg.String() == config.GetKeyPreview() {
			if m.varState != nil && m.varState.cheat != nil {
				if m.enterPreview(m.varState.cheat) {
					return tea.ClearScreen
				}
			}
		}
	}
	return nil
}

func (m *mainModel) completePathFromInput() bool {
	if m.varState == nil {
		return false
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

	show := len(result.Candidates) > 1
	m.varState.pathCompletions = result.Candidates
	m.varState.showPathCompletions = show
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
	m.varState.pathCompletions = nil
	m.varState.showPathCompletions = false
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
