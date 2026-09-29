package headless

import (
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"

	"github.com/cheatmd-dev/cheatmd/internal/resolver"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestResolvedSelectorColumn(t *testing.T) {
	previousShell := config.Get().Shell
	config.Get().Shell = "/bin/sh"
	t.Cleanup(func() { config.Get().Shell = previousShell })
	for _, tc := range []struct{ name, args, want string }{
		{"select column without map", "--delimiter : --select-column 1", "alpha"},
		{"select column with map", `--delimiter : --select-column 1 --map "printf mapped"`, "mapped"},
		{"display column only", "--delimiter : --column 2", "alpha:First"},
		{"no transform", "", "alpha:First"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &RunnerSession{}
			v := &resolver.VarState{Def: parser.VarDef{Args: tc.args}}
			s.applyResolvedValue(v, "alpha:First")
			if !v.Resolved || v.Value != tc.want {
				t.Fatalf("resolved=%v, value=%q, want resolved value %q", v.Resolved, v.Value, tc.want)
			}
		})
	}
}
