package headless

import (
	"reflect"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

func TestLiteralChainSeesEarlierResolvedValues(t *testing.T) {
	old := config.Get().VarSyntax
	t.Cleanup(func() { config.Get().VarSyntax = old })
	config.Get().VarSyntax = "dollar"
	for _, name := range []string{"a", "b", "c"} {
		t.Setenv(name, "")
	}
	cheat := &parser.Cheat{
		Command: "echo $c",
		Vars: []parser.VarDef{
			{Name: "a", Literal: "alpha"},
			{Name: "b", Literal: "$a-beta"},
			{Name: "c", Literal: "$b-gamma"},
		},
	}
	session := &RunnerSession{Cheat: cheat, Index: parser.NewCheatIndex()}
	session.initializeVariables()

	if err := session.resolveInteractively(); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"a": "alpha", "b": "alpha-beta", "c": "alpha-beta-gamma"}
	if !reflect.DeepEqual(cheat.Scope, want) {
		t.Fatalf("resolved scope = %v, want %v", cheat.Scope, want)
	}
}
