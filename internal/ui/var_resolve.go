package ui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cheatmd-dev/cheatmd/internal/resolver"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// ============================================================================
// Types
// ============================================================================

// shellResultMsg is sent when a shell command completes.
type shellResultMsg struct {
	options []string
	err     error
}

// ============================================================================
// Lifecycle
// ============================================================================

// startVarResolution initiates variable resolution and returns a command.
func (m *mainModel) startVarResolution() tea.Cmd {
	m.startVarResolutionInternal()
	if m.phase != phaseVarResolve {
		// No variables to resolve - finish immediately.
		return tea.Quit
	}
	return m.prepareCurrentVar()
}

// startVarResolutionInternal sets up variable resolution state.
func (m *mainModel) startVarResolutionInternal() {
	cheat := m.selected
	if cheat == nil {
		return
	}

	if cheat.Scope == nil {
		cheat.Scope = make(map[string]string)
	}

	vars := collectVariables(cheat, m.cheatIndex)
	if len(vars) == 0 {
		// No variables - stay in cheat select phase (will quit immediately).
		return
	}

	// Pre-fill from cheat.Scope (populated by --match) or environment.
	for i := range vars {
		varName := vars[i].def.Name
		if scopeVal, ok := cheat.Scope[varName]; ok && scopeVal != "" {
			vars[i].prefill = scopeVal
			vars[i].skipAutoCont = true
		} else if envVal := os.Getenv(varName); envVal != "" {
			vars[i].prefill = envVal
		}
		vars[i].multiSelectedSet = make(map[string]bool)
	}

	m.varState = &varResolveState{
		cheat:      cheat,
		vars:       vars,
		currentIdx: 0,
	}
	m.phase = phaseVarResolve

	// Save query and reset text input for variable resolution.
	m.lastQuery = m.textInput.Value()
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Type to filter or enter value..."
	m.picker.Cursor = 0
	m.picker.Offset = 0
}

// prepareCurrentVar prepares the current variable for display. May return a
// command to run a shell command to get options.
func (m *mainModel) prepareCurrentVar() tea.Cmd {
	if m.varState == nil || m.varState.currentIdx >= len(m.varState.vars) {
		if m.varState != nil {
			for _, vs := range m.varState.vars {
				if vs.resolved {
					m.selected.Scope[vs.def.Name] = vs.value
				}
			}
		}
		return tea.Quit
	}

	vs := &m.varState.vars[m.varState.currentIdx]
	scope := m.currentVarScope()

	selectedDef := selectVariant(vs.variants, scope)
	if selectedDef == nil {
		if allVariantsConditional(vs.variants) {
			vs.resolved = true
			vs.value = ""
			m.varState.currentIdx++
			return m.prepareCurrentVar()
		}
		selectedDef = &vs.def
	}
	vs.def = *selectedDef

	if config.Get().AutoContinue && vs.prefill != "" && !vs.skipAutoCont {
		vs.value = vs.prefill
		vs.resolved = true
		m.varState.currentIdx++
		return m.prepareCurrentVar()
	}

	m.varState.customHeader = extractCustomHeader(vs.def.Args)
	m.varState.selectOpts = resolver.ParseSelectorOpts(vs.def.Args)

	if vs.def.Literal != "" {
		return m.prepareLiteralVar(vs, scope)
	}

	if strings.TrimSpace(vs.def.Shell) == "" {
		return m.preparePromptVar(vs)
	}

	return m.prepareShellVar(vs, scope)
}

func (m *mainModel) currentVarScope() map[string]string {
	scope := make(map[string]string)
	for _, v := range m.varState.vars {
		if v.resolved {
			scope[v.def.Name] = v.value
		}
	}
	return scope
}

func allVariantsConditional(variants []parser.VarDef) bool {
	if len(variants) == 0 {
		return false
	}
	for _, v := range variants {
		if v.Condition == "" {
			return false
		}
	}
	return true
}

func (m *mainModel) prepareLiteralVar(vs *varState, scope map[string]string) tea.Cmd {
	result := executor.SubstituteVars(vs.def.Literal, scope, config.Get().VarSyntax)
	if vs.skipAutoCont {
		m.varState.isPromptOnly = true
		m.varState.options = nil
		if m.varState.picker != nil {
			m.varState.picker.SetItems(nil)
		}
		m.textInput.SetValue(result)
		m.textInput.CursorEnd()
		return nil
	}
	vs.value = result
	vs.resolved = true
	m.varState.currentIdx++
	return m.prepareCurrentVar()
}

func (m *mainModel) preparePromptVar(vs *varState) tea.Cmd {
	m.varState.isPromptOnly = true
	m.varState.options = nil
	if m.varState.picker != nil {
		m.varState.picker.SetItems(nil)
	}
	if vs.prefill != "" {
		m.textInput.SetValue(vs.prefill)
		m.textInput.CursorEnd()
	}
	return nil
}

func (m *mainModel) prepareShellVar(vs *varState, scope map[string]string) tea.Cmd {
	shellCmd := executor.SubstituteVars(vs.def.Shell, scope, config.Get().VarSyntax)
	return func() tea.Msg {
		output, err := m.executor.RunShell(shellCmd)
		if err != nil {
			return shellResultMsg{nil, err}
		}
		lines := parser.SplitLines(output)
		return shellResultMsg{lines, nil}
	}
}

// ============================================================================
// Update
// ============================================================================

// updateVarResolve handles updates during variable resolution phase.
func (m *mainModel) updateVarResolve(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cmd := m.handleVarResolveKey(msg); cmd != nil {
			return m, cmd
		}
	case shellResultMsg:
		return m.handleShellResult(msg)
	}

	if m.varState == nil {
		return m, nil
	}

	prevQuery := m.textInput.Value()
	var tiCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)

	if m.textInput.Value() != prevQuery {
		m.clearPathCompletions()
		if !m.varState.isPromptOnly && m.varState.picker != nil {
			m.varState.picker.Filter(m.textInput.Value())
		}
	}

	return m, tiCmd
}

// handleShellResult processes the result of a shell command.
func (m *mainModel) handleShellResult(msg shellResultMsg) (tea.Model, tea.Cmd) {
	if m.varState == nil {
		return m, nil
	}

	vs := &m.varState.vars[m.varState.currentIdx]

	if msg.err != nil {
		m.varState.shellErr = msg.err
		m.varState.isPromptOnly = true
		m.varState.options = nil
		if m.varState.picker != nil {
			m.varState.picker.SetItems(nil)
		}
		m.textInput.SetValue(vs.prefill)
		return m, nil
	}

	m.varState.options = msg.options
	m.varState.shellErr = nil

	switch len(msg.options) {
	case 0:
		m.varState.isPromptOnly = true
		if vs.prefill != "" {
			m.textInput.SetValue(vs.prefill)
			m.textInput.CursorEnd()
		}
	case 1:
		m.varState.isPromptOnly = true
		prefill := vs.prefill
		if prefill == "" {
			prefill = applyMapTransform(msg.options[0], m.varState.selectOpts)
		}
		m.textInput.SetValue(prefill)
		m.textInput.CursorEnd()
	default:
		m.varState.isPromptOnly = false

		// Build options list
		opts := m.varState.selectOpts
		items := make([]FilteredOption, len(msg.options))
		for i, opt := range msg.options {
			display := getDisplayColumn(opt, opts.Delimiter, opts.Column)
			items[i] = FilteredOption{
				Display:    display,
				Original:   opt,
				SearchText: strings.ToLower(display),
			}
		}

		if m.varState.picker == nil {
			m.varState.picker = NewPicker(items, func(opt FilteredOption, words []string) bool {
				return resolver.MatchesAllWords(opt.SearchText, words)
			})
		} else {
			m.varState.picker.SetItems(items)
		}
		if vs.prefill != "" {
			m.textInput.SetValue(vs.prefill)
			m.textInput.CursorEnd()
		}
		m.varState.picker.Filter(m.textInput.Value())

		// Set the cursor location, if the variable already exists in prefilled.
		if vs.prefill != "" {
			for i, opt := range m.varState.picker.Filtered {
				if opt.Original == vs.prefill {
					m.varState.picker.Cursor = i
					break
				}
			}
		}
	}

	return m, nil
}
