package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cheatmd-dev/cheatmd/internal/history"
	"github.com/cheatmd-dev/cheatmd/internal/resolver"
	"github.com/cheatmd-dev/cheatmd/pkg/chainstate"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// recordRun appends one entry to the history file. Errors are silently
// dropped; history is best-effort, never blocking execution.
func recordRun(cheat *parser.Cheat, finalCmd string) {
	if cheat == nil || finalCmd == "" {
		return
	}
	path, err := history.DefaultPath(config.Get().HistoryFile)
	if err != nil {
		return
	}
	// Copy scope so later mutations to cheat.Scope can't corrupt the entry.
	var scopeCopy map[string]string
	if len(cheat.Scope) > 0 {
		scopeCopy = make(map[string]string, len(cheat.Scope))
		for k, v := range cheat.Scope {
			scopeCopy[k] = v
		}
	}
	_ = history.Append(path, history.Entry{
		Command:	finalCmd,
		File:		cheat.File,
		Header:		cheat.Header,
		Scope:		scopeCopy,
	})
}

// getTTY returns file handles for TUI input/output. Uses /dev/tty to bypass
// shell pipes and command substitution when stdout is not a terminal.
func getTTY() (in *os.File, out *os.File, cleanup func()) {
	var closers []func()

	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		// stdout is NOT a terminal - we're being captured.
		out, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			out = os.Stderr
		} else {
			closers = append(closers, func() { out.Close() })
		}

		in, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
		if err != nil {
			in = os.Stdin
		} else {
			closers = append(closers, func() { in.Close() })
		}

		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(out))

		return in, out, func() {
			for _, c := range closers {
				c()
			}
		}
	}

	return os.Stdin, os.Stdout, func() {}
}

// RunTUI launches the Bubble Tea interface (unified, no flicker).
func RunTUI(index *parser.CheatIndex, exec Executor, initialQuery, matchCmd string) (string, error) {
	return RunTUIWithStart(index, exec, initialQuery, matchCmd, phaseCheatSelect)
}

// RunTUIWithStart launches the TUI and (optionally) jumps directly into a
// non-default starting phase. startPhase == phaseCheatSelect behaves the
// same as RunTUI; phaseHistory opens the history overlay on entry.
func RunTUIWithStart(index *parser.CheatIndex, exec Executor, initialQuery, matchCmd string, startPhase uiPhase) (string, error) {
	requireCheatBlock := config.Get().RequireCheatBlock
	cheats := index.FilterByConfig(requireCheatBlock)
	if len(cheats) == 0 {
		return "", fmt.Errorf("no cheats found")
	}

	m := newMainModel(cheats, index, exec)
	if scores := loadFrecencyScores(); len(scores) > 0 {
		m.applyFrecency(scores)
	}
	m.chainPath, _ = chainstate.DefaultPath()
	if m.chainPath != "" {
		m.chainState, _ = chainstate.Load(m.chainPath)
	}

	initialQuery, resumeChain := applyInitialChainState(&m, index, initialQuery, matchCmd, startPhase)

	if matchCmd != "" {
		if finalCmd, ok := handleMatchCmd(&m, index, exec, matchCmd); ok {
			return finalCmd, nil
		}
		if m.selected == nil {
			initialQuery = matchCmd
		}
	}

	if initialQuery != "" {
		if finalCmd, ok := handleInitialQuery(&m, index, exec, initialQuery, resumeChain); ok {
			return finalCmd, nil
		}
	}

	if startPhase == phaseHistory && m.selected == nil && initialQuery == "" {
		m.enterHistory()
	}

	in, out, cleanup := getTTY()
	defer cleanup()
	RefreshStyles()

	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(out))
	if _, err := p.Run(); err != nil {
		return "", err
	}

	if m.selected != nil {
		return finishCheat(&m, index, exec), nil
	}
	return "", nil
}

func applyInitialChainState(m *mainModel, index *parser.CheatIndex, initialQuery, matchCmd string, startPhase uiPhase) (string, bool) {
	if initialQuery == "" && matchCmd == "" && startPhase == phaseCheatSelect {
		if active := chainstate.ActiveName(index.Root, m.chainState); active != "" {
			return "/chain " + active, true
		}
	}
	return initialQuery, false
}

func handleMatchCmd(m *mainModel, index *parser.CheatIndex, exec Executor, matchCmd string) (string, bool) {
	matched := resolver.FindMatchingCheat(index.FilterByConfig(config.Get().RequireCheatBlock), matchCmd)
	if matched == nil {
		return "", false
	}
	m.selected = matched
	resolver.PrefillScopeFromMatch(matched, matchCmd)
	resolver.InferDependentVars(matched, index)
	m.startVarResolutionInternal()

	if m.phase != phaseVarResolve {
		return finishCheat(m, index, exec), true
	}

	if m.varState != nil && len(m.varState.vars) > 0 {
		if vs := &m.varState.vars[0]; vs.prefill != "" {
			m.textInput.SetValue(vs.prefill)
			m.textInput.CursorEnd()
		}
	}
	return "", false
}

func handleInitialQuery(m *mainModel, index *parser.CheatIndex, exec Executor, initialQuery string, resumeChain bool) (string, bool) {
	m.textInput.SetValue(initialQuery)
	m.filterCheats()

	if (config.Get().AutoSelect || resumeChain) && len(m.picker.Filtered) == 1 {
		m.selected = m.picker.Filtered[0].cheat
		m.startVarResolutionInternal()

		if m.phase != phaseVarResolve {
			return finishCheat(m, index, exec), true
		}

		if m.varState != nil && len(m.varState.vars) > 0 {
			if vs := &m.varState.vars[0]; vs.prefill != "" {
				m.textInput.SetValue(vs.prefill)
				m.textInput.CursorEnd()
			}
		}
	}
	return "", false
}

func loadFrecencyScores() map[string]float64 {
	path, err := history.DefaultPath(config.Get().HistoryFile)
	if err != nil {
		return nil
	}
	entries, err := history.Load(path, config.Get().HistoryMax)
	if err != nil {
		return nil
	}
	return history.FrecencyScores(entries, time.Now())
}

func advanceChain(index *parser.CheatIndex, cheat *parser.Cheat, path string, state *chainstate.State) {
	if cheat == nil || cheat.ChainName == "" || cheat.ChainStep < 1 || path == "" {
		return
	}
	if state == nil {
		state = &chainstate.State{Projects: make(map[string]*chainstate.ProjectState)}
	}

	maxStep := index.ChainMaxSteps[cheat.ChainName]
	next := cheat.ChainStep + 1

	if maxStep == 0 || next > maxStep {
		next = 1
		chainstate.SetActive(index.Root, "", state)
	} else {
		chainstate.SetActive(index.Root, cheat.ChainName, state)
	}
	chainstate.SetStep(index.Root, cheat.ChainName, next, state)
	_ = chainstate.Save(path, state)
}

// openFileInViewer opens filePath in the configured editor or system default.
func openFileInViewer(filePath string) {
	var cmd *exec.Cmd

	if editor := config.Get().Editor; editor != "" {
		cmd = exec.Command(editor, filePath)
	} else {
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", filePath)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", "", filePath)
		default:
			cmd = exec.Command("xdg-open", filePath)
		}
	}
	_ = cmd.Start()
}

func finishCheat(m *mainModel, index *parser.CheatIndex, exec Executor) string {
	finalCmd := exec.BuildFinalCommand(m.selected)
	recordRun(m.selected, finalCmd)
	advanceChain(index, m.selected, m.chainPath, m.chainState)
	return finalCmd
}
