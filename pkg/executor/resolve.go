package executor

import (
	"regexp"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// CollectDependencies gathers all variable definitions and their topological ordering.
func CollectDependencies(cheat *parser.Cheat, index *parser.CheatIndex) ([]string, map[string][]parser.VarDef) {
	varDefs := CollectVarDefinitions(cheat, index)
	usedVars := FindAllVars(cheat.Command, config.Get().VarSyntax)

	if config.Get().AllowUndeclaredVars {
		for _, name := range usedVars {
			if _, ok := varDefs[name]; !ok {
				varDefs[name] = []parser.VarDef{{Name: name}}
			}
		}
	}

	allNeeded := FindAllDependencies(usedVars, varDefs)
	orderedVars := TopologicalSort(usedVars, varDefs, allNeeded)

	return orderedVars, varDefs
}

// CollectVarDefinitions gathers all var definitions from imports and local cheat.
func CollectVarDefinitions(cheat *parser.Cheat, index *parser.CheatIndex) map[string][]parser.VarDef {
	varDefs := make(map[string][]parser.VarDef)

	var collectFromImports func(imports []string, seen map[string]bool)
	collectFromImports = func(imports []string, seen map[string]bool) {
		for _, importName := range imports {
			if seen[importName] {
				continue
			}
			seen[importName] = true
			module, ok := index.Modules[importName]
			if !ok {
				continue
			}
			collectFromImports(module.Imports, seen)
			for _, v := range module.Vars {
				varDefs[v.Name] = append(varDefs[v.Name], v)
			}
		}
	}
	collectFromImports(cheat.Imports, make(map[string]bool))

	for _, v := range cheat.Vars {
		varDefs[v.Name] = append(varDefs[v.Name], v)
	}
	return varDefs
}

func varDefDependencies(def parser.VarDef) []string {
	var deps []string
	deps = append(deps, FindAllVars(def.Shell, "dollar")...)
	deps = append(deps, FindAllVars(def.Literal, "dollar")...)
	deps = append(deps, FindAllVars(def.Condition, "dollar")...)
	return deps
}

// FindAllDependencies finds transitive closure of all needed variables.
func FindAllDependencies(usedVars []string, varDefs map[string][]parser.VarDef) map[string]bool {
	allNeeded := make(map[string]bool)
	queue := make([]string, len(usedVars))
	copy(queue, usedVars)

	for len(queue) > 0 {
		varName := queue[0]
		queue = queue[1:]

		if allNeeded[varName] {
			continue
		}
		allNeeded[varName] = true

		queue = enqueueDependencies(queue, varDefs[varName], allNeeded)
	}
	return allNeeded
}

func enqueueDependencies(queue []string, defs []parser.VarDef, allNeeded map[string]bool) []string {
	for _, def := range defs {
		for _, dep := range varDefDependencies(def) {
			if !allNeeded[dep] {
				queue = append(queue, dep)
			}
		}
	}
	return queue
}

// TopologicalSort orders variables by their dependencies.
func TopologicalSort(usedVars []string, varDefs map[string][]parser.VarDef, allNeeded map[string]bool) []string {
	var orderedVars []string
	added := make(map[string]bool)
	visiting := make(map[string]bool)

	var addWithDeps func(varName string)
	addWithDeps = func(varName string) {
		if added[varName] || !allNeeded[varName] || visiting[varName] {
			return
		}
		visiting[varName] = true
		visitDependencies(varDefs[varName], addWithDeps)
		visiting[varName] = false
		added[varName] = true
		orderedVars = append(orderedVars, varName)
	}

	for _, v := range usedVars {
		addWithDeps(v)
	}
	return orderedVars
}

func visitDependencies(defs []parser.VarDef, addWithDeps func(string)) {
	for _, def := range defs {
		for _, dep := range varDefDependencies(def) {
			addWithDeps(dep)
		}
	}
}

// EvaluateCondition evaluates a condition expression against the scope.
func EvaluateCondition(condition string, scope map[string]string) bool {
	condition = strings.TrimSpace(condition)

	condition = SubstituteVars(condition, scope, "dollar")

	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left == right
		}
	}

	if strings.Contains(condition, "!=") {
		parts := strings.SplitN(condition, "!=", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left != right
		}
	}

	return condition != ""
}

// ReplaceVar replaces variable references in cmd with replacement.
func ReplaceVar(cmd, varName, replacement string, syntax string) string {
	q := regexp.QuoteMeta(varName)
	var parts []string
	if syntax == "dollar" || syntax == "both" {
		parts = append(parts, `\$`+q+`\b`)
	}
	if syntax == "angle" || syntax == "both" {
		parts = append(parts, `<`+q+`>`)
	}
	if len(parts) == 0 {
		return cmd
	}
	pattern := strings.Join(parts, "|")
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllLiteralString(cmd, replacement)
}

// FindAllVars finds variable references using the parser's syntax and escaping rules.
func FindAllVars(cmd string, syntax string) []string {
	return parser.ExtractVars(cmd, syntax == "" || syntax == "dollar" || syntax == "both", syntax == "angle" || syntax == "both")
}
