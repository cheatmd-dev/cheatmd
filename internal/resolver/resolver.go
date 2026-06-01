// Package resolver acts as a bridge connecting the parsed AST to the UI and Executor engines.
// It provides functionality for fuzzy matching queries, determining variable dependencies,
// and assembling the topological execution graph.
package resolver

import (
	"fmt"

	"github.com/cheatmd-dev/cheatmd/pkg/executor"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// SelectOptions holds display and extraction options for selectors.
type SelectOptions struct {
	Delimiter    string
	Column       int    // 1-indexed, 0 = all (display column)
	SelectColumn int    // 1-indexed, 0 = no extraction (return full/original line)
	MapCmd       string // command to transform selected value
	Multi        bool   // true if --multi is provided
}

// VarState tracks a variable and its resolved value during resolution progress.
type VarState struct {
	Def              parser.VarDef   // The active definition
	Variants         []parser.VarDef // All conditional variants (for if/fi blocks)
	Value            string
	Resolved         bool
	Prefill          string
	SkipAutoCont     bool // True if user went back to this var - don't auto-continue
	MultiSelected    []string
	MultiSelectedSet map[string]bool
}

// CollectVariables gathers all variable definitions from imports and local
// and wraps them in exported VarState objects.
func CollectVariables(cheat *parser.Cheat, index *parser.CheatIndex) []VarState {
	orderedVars, varDefs := executor.CollectDependencies(cheat, index)

	var vars []VarState
	for _, varName := range orderedVars {
		if defs, ok := varDefs[varName]; ok && len(defs) > 0 {
			vars = append(vars, VarState{
				Def:      defs[0],
				Variants: defs,
			})
		}
	}
	return vars
}

// SelectVariant picks the first variant whose condition matches, or nil if none match.
// Returns the first unconditional variant as fallback (default case).
func SelectVariant(variants []parser.VarDef, scope map[string]string) *parser.VarDef {
	var defaultDef *parser.VarDef

	for i := range variants {
		v := &variants[i]
		if v.Condition == "" {
			// Unconditional - this is the default/fallback
			if defaultDef == nil {
				defaultDef = v
			}
			continue
		}
		if executor.EvaluateCondition(v.Condition, scope) {
			return v
		}
	}

	return defaultDef
}

// ExtractCustomHeader parses --header from selector arguments.
func ExtractCustomHeader(selectorArgs string) string {
	if selectorArgs == "" {
		return ""
	}
	args := ParseShellArgs(selectorArgs)
	for i := 0; i < len(args); i++ {
		if args[i] == "--header" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// ParseSelectorOpts parses selector options from arguments.
func ParseSelectorOpts(selectorArgs string) SelectOptions {
	opts := SelectOptions{}
	if selectorArgs == "" {
		return opts
	}

	args := ParseShellArgs(selectorArgs)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--delimiter":
			if i+1 < len(args) {
				opts.Delimiter = args[i+1]
				i++
			}
		case "--column":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.Column)
				i++
			}
		case "--select-column":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opts.SelectColumn)
				i++
			}
		case "--map":
			if i+1 < len(args) {
				opts.MapCmd = args[i+1]
				i++
			}
		case "--multi":
			opts.Multi = true
		}
	}
	return opts
}
