package executor

import (
	"reflect"
	"testing"
)

func TestFindAllVarsEscapesAndDefaultSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, command, syntax string
		want                  []string
	}{
		{"default", "echo $value", "", []string{"value"}},
		{"odd backslash", `echo \$literal $value`, "dollar", []string{"value"}},
		{"even backslashes", `echo \\$value`, "dollar", nil},
		{"three backslashes", `echo \\\$literal $value`, "dollar", []string{"value"}},
		{"shell braces", "echo ${HOME} $value", "dollar", []string{"value"}},
		{"ordered unique names", "echo $first <second> $first", "both", []string{"first", "second"}},
		{"escaped and active references", `echo \$value \\$literal $value`, "dollar", []string{"value"}},
		{"angle only", "echo $first <second>", "angle", []string{"second"}},
		{"malformed angle", "echo <broken $value", "both", []string{"value"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FindAllVars(tc.command, tc.syntax); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("FindAllVars() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReplaceVarPreservesEscapedReference(t *testing.T) {
	input := `echo \$name \\$name $name <name> $names`
	want := `echo \$name \\$name value value $names`
	if got := ReplaceVar(input, "name", "value", "both"); got != want {
		t.Fatalf("ReplaceVar = %q, want %q", got, want)
	}
}
