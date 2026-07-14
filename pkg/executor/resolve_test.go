package executor

import (
	"reflect"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func setVarSyntax(t *testing.T, syntax string) {
	t.Helper()
	oldConfig := *config.Get()
	config.Get().VarSyntax = syntax
	t.Cleanup(func() {
		*config.Get() = oldConfig
	})
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		scope     map[string]string
		expected  bool
	}{
		{
			name:      "simple equality match",
			condition: "$status == ok",
			scope:     map[string]string{"status": "ok"},
			expected:  true,
		},
		{
			name:      "simple equality mismatch",
			condition: "$status == ok",
			scope:     map[string]string{"status": "error"},
			expected:  false,
		},
		{
			name:      "inequality match",
			condition: "$status != ok",
			scope:     map[string]string{"status": "error"},
			expected:  true,
		},
		{
			name:      "inequality mismatch",
			condition: "$status != ok",
			scope:     map[string]string{"status": "ok"},
			expected:  false,
		},
		{
			name:      "unconditional true",
			condition: "true",
			scope:     map[string]string{},
			expected:  true,
		},
		{
			name:      "empty condition is false",
			condition: "",
			scope:     map[string]string{},
			expected:  false,
		},
		{
			name:      "whitespace handling",
			condition: "  $env   ==   prod  ",
			scope:     map[string]string{"env": "prod"},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCondition(tt.condition, tt.scope)
			if got != tt.expected {
				t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.condition, got, tt.expected)
			}
		})
	}
}

func TestFindAllVars(t *testing.T) {
	tests := []struct {
		name     string
		syntax   string
		cmd      string
		expected []string
	}{
		{
			name:     "dollar vars",
			syntax:   "dollar",
			cmd:      "echo $foo and $bar",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "angle vars",
			syntax:   "angle",
			cmd:      "echo <foo> and <bar>",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "both vars",
			syntax:   "both",
			cmd:      "echo $foo and <bar>",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "escaped dollar ignored",
			syntax:   "dollar",
			cmd:      "echo \\$foo and $bar",
			expected: []string{"bar"},
		},
		{
			name:     "deduplication",
			syntax:   "dollar",
			cmd:      "echo $foo and $foo",
			expected: []string{"foo"},
		},
		{
			name:     "braced shell vars ignored",
			syntax:   "dollar",
			cmd:      "echo ${foo} $bar",
			expected: []string{"bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindAllVars(tt.cmd, tt.syntax)
			if len(got) == 0 && len(tt.expected) == 0 {
				return // Both empty
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("FindAllVars(%q, %q) = %v, want %v", tt.cmd, tt.syntax, got, tt.expected)
			}
		})
	}
}

func TestCollectDependencies(t *testing.T) {
	setVarSyntax(t, "dollar")

	index := parser.NewCheatIndex()
	cheat := &parser.Cheat{
		Command: "echo $a $b",
		Vars: []parser.VarDef{
			{Name: "a", Shell: "echo $c"},
			{Name: "b", Shell: "echo $d"},
			{Name: "c", Literal: "hello"},
			{Name: "d", Shell: "echo $c"},
		},
	}

	orderedVars, varDefs := CollectDependencies(cheat, index)

	// Note: Topological sort order can sometimes vary depending on map iteration if paths are parallel,
	// but here c must be before a and d, and d must be before b. a and d can be swapped.
	// We'll check constraints instead of exact order to avoid flakiness.

	if len(orderedVars) != 4 {
		t.Fatalf("expected 4 ordered vars, got %v", orderedVars)
	}

	pos := make(map[string]int)
	for i, v := range orderedVars {
		pos[v] = i
	}

	if pos["c"] > pos["a"] {
		t.Errorf("expected 'c' to be resolved before 'a'")
	}
	if pos["c"] > pos["d"] {
		t.Errorf("expected 'c' to be resolved before 'd'")
	}
	if pos["d"] > pos["b"] {
		t.Errorf("expected 'd' to be resolved before 'b'")
	}

	if len(varDefs) != 4 {
		t.Errorf("expected 4 var definitions, got %v", varDefs)
	}
}

func TestCollectDependencies_Undeclared(t *testing.T) {
	setVarSyntax(t, "dollar")
	oldConfig := *config.Get()
	config.Get().AllowUndeclaredVars = true
	defer func() { *config.Get() = oldConfig }()

	index := parser.NewCheatIndex()
	cheat := &parser.Cheat{
		Command: "echo $undeclared",
		Vars:    []parser.VarDef{},
	}

	orderedVars, varDefs := CollectDependencies(cheat, index)
	if len(orderedVars) != 1 || orderedVars[0] != "undeclared" {
		t.Errorf("expected ['undeclared'], got %v", orderedVars)
	}
	if _, ok := varDefs["undeclared"]; !ok {
		t.Errorf("expected 'undeclared' in varDefs")
	}
}
