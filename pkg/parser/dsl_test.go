package parser

import (
	"testing"
)

func TestDSLIfNesting(t *testing.T) {
	cheat := &Cheat{}
	dslBlock := `
if A
	if B
		var X = echo x
	fi
	var Y = echo y
fi
`
	errs := parseCheatDSL(cheat, dslBlock, "test.md", 1)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(cheat.Vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(cheat.Vars))
	}

	for _, v := range cheat.Vars {
		if v.Name == "X" && v.Condition != "A && B" {
			t.Errorf("expected var X condition to be A && B, got %s", v.Condition)
		}
		if v.Name == "Y" && v.Condition != "A" {
			t.Errorf("expected var Y condition to be A, got %s", v.Condition)
		}
	}
}
