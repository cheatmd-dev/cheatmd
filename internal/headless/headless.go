package headless

import (
	"bufio"
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
	Scanner *bufio.Scanner
}

// Run acts as the primary entry point, constructing a session and starting the execution.
func Run(index *parser.CheatIndex, exec Executor, initialQuery, matchCmd string) error {
	session := &RunnerSession{
		Index:   index,
		Exec:    exec,
		Scanner: bufio.NewScanner(os.Stdin),
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
	cheats := s.Index.FilterByConfig(config.GetRequireCheatBlock())
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

// initializeVariables extracts and pre-fills environmental and scoped variables,
// priming the parameter list for resolution.
func (s *RunnerSession) initializeVariables() {
	if s.Cheat.Scope == nil {
		s.Cheat.Scope = make(map[string]string)
	}

	s.Vars = resolver.CollectVariables(s.Cheat, s.Index)
	for i := range s.Vars {
		s.primeVariable(&s.Vars[i])
	}
}

func (s *RunnerSession) primeVariable(vs *resolver.VarState) {
	varName := vs.Def.Name
	vs.MultiSelectedSet = make(map[string]bool)

	if scopeVal, ok := s.Cheat.Scope[varName]; ok && scopeVal != "" {
		vs.Prefill = scopeVal
		vs.SkipAutoCont = true
		return
	}
	if envVal := os.Getenv(varName); envVal != "" {
		vs.Prefill = envVal
	}
}

// resolveInteractively loops through the state machine, attempting auto-resolution
// before falling back to prompting the user over JSON-RPC until all variables are resolved.
func (s *RunnerSession) resolveInteractively() error {
	for !s.allVariablesResolved() {
		s.autoResolveLoop()
		if s.allVariablesResolved() {
			break
		}

		if err := s.promptNextBatch(); err != nil {
			return err
		}
	}

	s.commitFinalizedVariables()
	return nil
}

func (s *RunnerSession) promptNextBatch() error {
	promptVars, err := s.collectUnresolvedDependencies()
	if err != nil {
		return err
	}
	return s.promptClient(promptVars)
}

func (s *RunnerSession) commitFinalizedVariables() {
	for _, vs := range s.Vars {
		if vs.Resolved {
			s.Cheat.Scope[vs.Def.Name] = vs.Value
		}
	}
}

// autoResolveLoop continuously evaluates conditions and literal dependencies,
// resolving variable states automatically whenever possible without user interaction.
func (s *RunnerSession) autoResolveLoop() {
	for s.attemptAutoResolvePass() {
	}
}

func (s *RunnerSession) attemptAutoResolvePass() bool {
	progress := false
	scope := s.buildCurrentScope()

	for i := range s.Vars {
		if s.tryResolveVariable(&s.Vars[i], scope) {
			progress = true
		}
	}

	return progress
}

func (s *RunnerSession) tryResolveVariable(vs *resolver.VarState, scope map[string]string) bool {
	if vs.Resolved {
		return false
	}
	if !s.areConditionDependenciesResolved(vs) {
		return false
	}

	s.updateVariableDefinition(vs, scope)

	if s.tryAutoContinue(vs) {
		return true
	}

	return s.tryLiteralResolution(vs, scope)
}

func (s *RunnerSession) updateVariableDefinition(vs *resolver.VarState, scope map[string]string) {
	selectedDef := resolver.SelectVariant(vs.Variants, scope)
	if selectedDef != nil {
		vs.Def = *selectedDef
		return
	}

	if s.allVariantsConditional(vs) {
		vs.Resolved = true
		vs.Value = ""
	}
}

func (s *RunnerSession) tryAutoContinue(vs *resolver.VarState) bool {
	if !s.canAutoContinue(vs) {
		return false
	}
	vs.Value = vs.Prefill
	vs.Resolved = true
	return true
}

func (s *RunnerSession) tryLiteralResolution(vs *resolver.VarState, scope map[string]string) bool {
	if vs.Def.Literal == "" {
		return false
	}
	if !s.areLiteralDependenciesResolved(vs.Def.Literal) {
		return false
	}
	vs.Value = executor.SubstituteVars(vs.Def.Literal, scope, "dollar")
	vs.Resolved = true
	return true
}

// buildCurrentScope generates a temporary evaluation context representing
// all currently resolved variables.
func (s *RunnerSession) buildCurrentScope() map[string]string {
	scope := make(map[string]string)
	for _, v := range s.Vars {
		if v.Resolved {
			scope[v.Def.Name] = v.Value
		}
	}
	return scope
}

// allVariablesResolved determines if all variables are fully resolved.
func (s *RunnerSession) allVariablesResolved() bool {
	for _, v := range s.Vars {
		if !v.Resolved {
			return false
		}
	}
	return true
}

// areConditionDependenciesResolved verifies that any variables required to evaluate
// the conditions of this variable's variants have already been resolved.
func (s *RunnerSession) areConditionDependenciesResolved(vs *resolver.VarState) bool {
	for _, variant := range vs.Variants {
		if !s.isVariantConditionResolved(variant) {
			return false
		}
	}
	return true
}

func (s *RunnerSession) isVariantConditionResolved(variant parser.VarDef) bool {
	if variant.Condition == "" {
		return true
	}
	deps := executor.FindAllVars(variant.Condition, "dollar")
	return s.areDependenciesResolved(deps)
}

func (s *RunnerSession) areDependenciesResolved(deps []string) bool {
	for _, dep := range deps {
		if !s.isDependencyResolved(dep) {
			return false
		}
	}
	return true
}

// isDependencyResolved checks the resolution state of a single dependency.
func (s *RunnerSession) isDependencyResolved(depName string) bool {
	for _, ov := range s.Vars {
		if ov.Def.Name == depName && ov.Resolved {
			return true
		}
	}
	return false
}

// allVariantsConditional determines if a variable purely consists of conditional variants
// allowing it to safely resolve to empty if no conditions match.
func (s *RunnerSession) allVariantsConditional(vs *resolver.VarState) bool {
	allConditional := true
	for _, v := range vs.Variants {
		if v.Condition == "" {
			allConditional = false
			break
		}
	}
	return allConditional && len(vs.Variants) > 0
}

// canAutoContinue enforces the configuration rules for automatically advancing
// through prefilled fields without prompting.
func (s *RunnerSession) canAutoContinue(vs *resolver.VarState) bool {
	autoContinue := config.GetAutoContinue()
	return autoContinue && vs.Prefill != "" && !vs.SkipAutoCont
}

// areLiteralDependenciesResolved ensures that a literal parameter transformation
// has all required dependencies before attempting evaluation.
func (s *RunnerSession) areLiteralDependenciesResolved(literal string) bool {
	deps := executor.FindAllVars(literal, "dollar")
	return s.areDependenciesResolved(deps)
}

// collectUnresolvedDependencies compiles the next batch of variables requiring user input,
// evaluating any dynamic shell scripts for option enumeration.
func (s *RunnerSession) collectUnresolvedDependencies() ([]promptVar, error) {
	var promptVars []promptVar
	scope := s.buildCurrentScope()

	for i := range s.Vars {
		pv := s.buildPromptVarIfReady(&s.Vars[i], i, scope)
		if pv != nil {
			promptVars = append(promptVars, *pv)
		}
	}

	if len(promptVars) == 0 {
		return nil, fmt.Errorf("resolution deadlock: cyclical dependencies detected, resolution stopped")
	}

	return promptVars, nil
}

func (s *RunnerSession) buildPromptVarIfReady(vs *resolver.VarState, idx int, scope map[string]string) *promptVar {
	if vs.Resolved {
		return nil
	}
	if vs.Def.Literal != "" {
		return nil
	}
	if !s.areConditionDependenciesResolved(vs) {
		return nil
	}

	selectOpts := resolver.ParseSelectorOpts(vs.Def.Args)
	options := s.evaluateShellOptions(vs, scope, selectOpts)

	return &promptVar{
		Name:        vs.Def.Name,
		Header:      resolver.ExtractCustomHeader(vs.Def.Args),
		Placeholder: vs.Prefill,
		Options:     options,
		Multi:       selectOpts.Multi,
		varIdx:      idx,
	}
}

func (s *RunnerSession) evaluateShellOptions(vs *resolver.VarState, scope map[string]string, selectOpts resolver.SelectOptions) []string {
	if strings.TrimSpace(vs.Def.Shell) == "" {
		return nil
	}

	shellCmd := executor.SubstituteVars(vs.Def.Shell, scope, "dollar")
	output, err := s.Exec.RunShell(shellCmd)
	if err != nil {
		return nil
	}

	return s.parseShellOptions(output, selectOpts)
}

func (s *RunnerSession) parseShellOptions(output string, selectOpts resolver.SelectOptions) []string {
	var options []string
	lines := parser.SplitLines(output)
	for _, opt := range lines {
		display := resolver.GetDisplayColumn(opt, selectOpts.Delimiter, selectOpts.Column)
		options = append(options, display)
	}
	return options
}

// promptClient sends a JSON-RPC request over the wire, requesting the user
// to manually supply the required variable values.
func (s *RunnerSession) promptClient(promptVars []promptVar) error {
	reqBytes, err := s.marshalPromptRequest(promptVars)
	if err != nil {
		return err
	}

	fmt.Println(string(reqBytes))

	if !s.Scanner.Scan() {
		return fmt.Errorf("client connection severed unexpectedly during variable prompt")
	}

	promptRes, err := s.parsePromptResponse(s.Scanner.Text())
	if err != nil {
		return err
	}

	s.ingestPromptValues(promptVars, promptRes)
	return nil
}

func (s *RunnerSession) marshalPromptRequest(promptVars []promptVar) ([]byte, error) {
	promptReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "prompt",
		"params": map[string]interface{}{
			"variables": promptVars,
		},
		"id": 1,
	}

	reqBytes, err := json.Marshal(promptReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt request: %w", err)
	}
	return reqBytes, nil
}

type promptResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Values map[string]string `json:"values"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Id int `json:"id"`
}

func (s *RunnerSession) parsePromptResponse(line string) (*promptResponse, error) {
	var promptRes promptResponse
	if err := json.Unmarshal([]byte(line), &promptRes); err != nil {
		return nil, fmt.Errorf("failed to parse client prompt response: %w", err)
	}

	if promptRes.Error != nil {
		return nil, fmt.Errorf("client aborted prompt: %s", promptRes.Error.Message)
	}

	return &promptRes, nil
}

func (s *RunnerSession) ingestPromptValues(promptVars []promptVar, promptRes *promptResponse) {
	for _, pv := range promptVars {
		val := s.extractPromptValue(pv, promptRes)
		s.applyResolvedValue(&s.Vars[pv.varIdx], val)
	}
}

func (s *RunnerSession) extractPromptValue(pv promptVar, promptRes *promptResponse) string {
	val, ok := promptRes.Result.Values[pv.Name]
	if !ok {
		return pv.Placeholder
	}
	return val
}

func (s *RunnerSession) applyResolvedValue(vs *resolver.VarState, val string) {
	selectOpts := resolver.ParseSelectorOpts(vs.Def.Args)
	if selectOpts.MapCmd != "" {
		val = resolver.ApplyMapTransform(val, selectOpts)
	}

	vs.Value = val
	vs.Resolved = true
}

// runCommand constructs the final command string, attaches any configured hooks,
// executes the command on the target shell, and reports the output via JSON-RPC.
func (s *RunnerSession) runCommand() error {
	finalCmd := s.buildAndRecordCommand()
	runErr, stdout, stderr := s.executeWithConfiguredMode(finalCmd)
	return s.reportCompletion(finalCmd, stdout, stderr, runErr)
}

func (s *RunnerSession) buildAndRecordCommand() string {
	finalCmd := s.Exec.BuildFinalCommand(s.Cheat)

	if preHook := config.GetPreHook(); preHook != "" {
		finalCmd = preHook + finalCmd
	}
	if postHook := config.GetPostHook(); postHook != "" {
		finalCmd = finalCmd + postHook
	}
	return finalCmd
}

func (s *RunnerSession) executeWithConfiguredMode(finalCmd string) (error, string, string) {
	switch config.GetOutput() {
	case "exec":
		stdout, stderr, err := runCommandAndCapture(config.GetShell(), finalCmd)
		return err, stdout, stderr
	case "copy":
		err := s.Exec.OutputWithMode(finalCmd, executor.OutputCopy)
		return err, "", ""
	default: // print
		return nil, finalCmd, ""
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
