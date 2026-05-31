package headless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cheatmd-dev/cheatmd/internal/resolver"
	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// Executor defines the interface required by the headless runner for shell command execution.
type Executor interface {
	RunShell(command string) (string, error)
	BuildFinalCommand(cheat *parser.Cheat) string
	OutputWithMode(command string, mode executor.OutputMode) error
}

// promptVar defines the JSON-RPC structure for prompting variables.
type promptVar struct {
	Name        string   `json:"name"`
	Header      string   `json:"header"`
	Placeholder string   `json:"placeholder"`
	Options     []string `json:"options"`
	Multi       bool     `json:"multi"`
	varIdx      int      `json:"-"`
}

// RunnerSession encapsulates the state and lifecycle of a command execution process.
// It orchestrates variable resolution and final command execution.
type RunnerSession struct {
	Index   *parser.CheatIndex
	Exec    Executor
	Cheat   *parser.Cheat
	Vars    []resolver.VarState
	Decoder *json.Decoder
}

// Run acts as the primary entry point, constructing a session and starting the execution.
func Run(index *parser.CheatIndex, exec Executor, initialQuery, matchCmd string) error {
	session := &RunnerSession{
		Index:   index,
		Exec:    exec,
		Decoder: json.NewDecoder(os.Stdin),
	}
	return session.Execute(initialQuery, matchCmd)
}

// Execute initiates the command execution lifecycle. It locates the target cheat,
// resolves required variables, and executes the finalized command.
func (s *RunnerSession) Execute(initialQuery, matchCmd string) error {
	if err := s.findTargetCheat(initialQuery, matchCmd); err != nil {
		return err
	}
	s.initializeVariables()
	if err := s.resolveInteractively(); err != nil {
		return err
	}
	return s.runCommand()
}

// findTargetCheat scans the provided index for the optimal cheat block to run.
func (s *RunnerSession) findTargetCheat(initialQuery, matchCmd string) error {
	cheats := s.Index.FilterByConfig(config.Get().RequireCheatBlock)
	if len(cheats) == 0 {
		return fmt.Errorf("no executable cheats found in index")
	}

	if s.tryMatchByCommand(cheats, matchCmd) {
		return nil
	}

	query := s.resolveInitialQuery(initialQuery, matchCmd)
	if s.tryMatchByQuery(cheats, query) {
		return nil
	}

	return fmt.Errorf("headless runner requires a precise query or match command to isolate a single cheat block")
}

func (s *RunnerSession) resolveInitialQuery(initialQuery, matchCmd string) string {
	if matchCmd != "" {
		return matchCmd
	}
	return initialQuery
}

func (s *RunnerSession) tryMatchByCommand(cheats []*parser.Cheat, matchCmd string) bool {
	if matchCmd == "" {
		return false
	}
	s.Cheat = resolver.FindMatchingCheat(cheats, matchCmd)
	if s.Cheat == nil {
		return false
	}
	resolver.PrefillScopeFromMatch(s.Cheat, matchCmd)
	resolver.InferDependentVars(s.Cheat, s.Index)
	return true
}

func (s *RunnerSession) tryMatchByQuery(cheats []*parser.Cheat, query string) bool {
	if query == "" {
		return false
	}
	words := strings.Fields(strings.ToLower(query))
	matchedCheats := s.filterCheatsByWords(cheats, words)
	s.Cheat = s.findExactHeaderMatch(matchedCheats, query)
	if s.Cheat != nil {
		return true
	}
	if len(matchedCheats) > 0 {
		s.Cheat = matchedCheats[0]
		return true
	}
	return false
}

func (s *RunnerSession) filterCheatsByWords(cheats []*parser.Cheat, words []string) []*parser.Cheat {
	var matched []*parser.Cheat
	for _, c := range cheats {
		if cheatMatchesQuery(c, words) {
			matched = append(matched, c)
		}
	}
	return matched
}

func (s *RunnerSession) findExactHeaderMatch(cheats []*parser.Cheat, query string) *parser.Cheat {
	for _, mc := range cheats {
		if strings.EqualFold(mc.Header, query) {
			return mc
		}
	}
	return nil
}

// runCommand constructs the final command string, attaches any configured hooks,
// executes the command on the target shell, and reports the output via JSON-RPC.
func (s *RunnerSession) runCommand() error {
	finalCmd := s.buildAndRecordCommand()
	stdout, stderr, runErr := s.executeWithConfiguredMode(finalCmd)
	return s.reportCompletion(finalCmd, stdout, stderr, runErr)
}

func (s *RunnerSession) buildAndRecordCommand() string {
	finalCmd := s.Exec.BuildFinalCommand(s.Cheat)

	if preHook := config.Get().PreHook; preHook != "" {
		finalCmd = preHook + finalCmd
	}
	if postHook := config.Get().PostHook; postHook != "" {
		finalCmd = finalCmd + postHook
	}
	return finalCmd
}

func (s *RunnerSession) executeWithConfiguredMode(finalCmd string) (string, string, error) {
	switch config.Get().Output {
	case "exec":
		stdout, stderr, err := runCommandAndCapture(config.Get().Shell, finalCmd)
		return stdout, stderr, err
	case "copy":
		err := s.Exec.OutputWithMode(finalCmd, executor.OutputCopy)
		return "", "", err
	default: // print
		return finalCmd, "", nil
	}
}

func (s *RunnerSession) reportCompletion(finalCmd, stdout, stderr string, runErr error) error {
	status, errMsg := s.determineRunStatus(runErr)

	completedFrame := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "completed",
		"params": map[string]interface{}{
			"status":    status,
			"command":   finalCmd,
			"stdout":    stdout,
			"stderr":    stderr,
			"error":     errMsg,
			"exit_code": getExitCode(runErr),
		},
	}

	resBytes, err := json.Marshal(completedFrame)
	if err != nil {
		return fmt.Errorf("failed to encode completion output: %w", err)
	}
	fmt.Println(string(resBytes))

	return runErr
}

func (s *RunnerSession) determineRunStatus(err error) (string, string) {
	if err != nil {
		return "error", err.Error()
	}
	return "success", ""
}

// Helper Utilities
// -----------------------------------------------------------------------------

// runCommandAndCapture shells out the given command and intercepts both standard streams.
func runCommandAndCapture(shell, command string) (string, string, error) {
	cmd := exec.Command(shell, "-c", command)
	cmd.Env = os.Environ()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// getExitCode extracts the OS-level exit code from an execution error, or -1 if unavailable.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -1
}

// cheatMatchesQuery performs a heuristic search across the cheat's metadata for targeting.
func cheatMatchesQuery(cheat *parser.Cheat, words []string) bool {
	for _, word := range words {
		if !wordMatchesCheat(cheat, word) {
			return false
		}
	}
	return true
}

func wordMatchesCheat(cheat *parser.Cheat, word string) bool {
	if matchesBasicMetadata(cheat, word) {
		return true
	}
	return matchesAnyTag(cheat.Tags, word)
}

func matchesBasicMetadata(cheat *parser.Cheat, word string) bool {
	folder := strings.ToLower(filepath.Base(cheat.File))
	file := strings.ToLower(strings.TrimSuffix(filepath.Base(cheat.File), ".md"))
	header := strings.ToLower(cheat.Header)
	desc := strings.ToLower(cheat.Description)
	command := strings.ToLower(cheat.Command)

	return strings.Contains(folder, word) ||
		strings.Contains(file, word) ||
		strings.Contains(header, word) ||
		strings.Contains(desc, word) ||
		strings.Contains(command, word)
}

func matchesAnyTag(tags []string, word string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), word) {
			return true
		}
	}
	return false
}
