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
		{"even backslashes", `echo \\$value`, "dollar", []string{"value"}},
		{"three backslashes", `echo \\\$literal $value`, "dollar", []string{"value"}},
		{"shell braces", "echo ${HOME} $value", "dollar", []string{"value"}},
		{"ordered unique names", "echo $first <second> $first", "both", []string{"first", "second"}},
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
