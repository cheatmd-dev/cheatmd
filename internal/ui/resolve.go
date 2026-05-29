package ui

import (
	"github.com/gubarz/cheatmd/internal/resolver"
	"github.com/gubarz/cheatmd/pkg/parser"
)

// ============================================================================
// Entry Point
// ============================================================================

// Run launches the Bubble Tea TUI interface
func Run(index *parser.CheatIndex, exec Executor, initialQuery, matchCmd string) (string, error) {
	return RunTUI(index, exec, initialQuery, matchCmd)
}

// RunHistory launches the TUI with the history overlay open. If history is
// empty or unreadable, an error is returned without entering the TUI.
func RunHistory(index *parser.CheatIndex, exec Executor) (string, error) {
	return RunTUIWithStart(index, exec, "", "", phaseHistory)
}

// ============================================================================
// Variable Resolution
// ============================================================================

// SelectOptions holds display options for selection. Aliased to resolver options.
type SelectOptions = resolver.SelectOptions

// varState tracks a variable and its resolved value.
type varState struct {
	def              parser.VarDef   // The selected/active definition
	variants         []parser.VarDef // All conditional variants (for if/fi blocks)
	value            string
	resolved         bool
	prefill          string
	skipAutoCont     bool // True if user went back to this var - don't auto-continue
	multiSelected    []string
	multiSelectedSet map[string]bool
}

// collectVariables gathers all variable definitions from imports and local
// and wraps them in UI-specific varState objects.
func collectVariables(cheat *parser.Cheat, index *parser.CheatIndex) []varState {
	resVars := resolver.CollectVariables(cheat, index)

	vars := make([]varState, len(resVars))
	for i, rv := range resVars {
		vars[i] = varState{
			def:              rv.Def,
			variants:         rv.Variants,
			value:            rv.Value,
			resolved:         rv.Resolved,
			prefill:          rv.Prefill,
			skipAutoCont:     rv.SkipAutoCont,
			multiSelected:    rv.MultiSelected,
			multiSelectedSet: rv.MultiSelectedSet,
		}
	}
	return vars
}

// selectVariant picks the first variant whose condition matches, or nil if none match
func selectVariant(variants []parser.VarDef, scope map[string]string) *parser.VarDef {
	return resolver.SelectVariant(variants, scope)
}

// extractCustomHeader parses --header from selector args
func extractCustomHeader(selectorArgs string) string {
	return resolver.ExtractCustomHeader(selectorArgs)
}

// parseShellArgs parses a string into arguments, respecting quotes
func parseShellArgs(s string) []string {
	return resolver.ParseShellArgs(s)
}

// applyMapTransform transforms the selected value based on options
func applyMapTransform(value string, opts SelectOptions) string {
	return resolver.ApplyMapTransform(value, opts)
}

// getDisplayColumn extracts the display column from a line
func getDisplayColumn(line, delimiter string, column int) string {
	return resolver.GetDisplayColumn(line, delimiter, column)
}
