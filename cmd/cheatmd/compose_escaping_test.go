package main

import (
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

func TestComposeOnlyRewritesVariableReferences(t *testing.T) {
	previous := config.Get().VarSyntax
	t.Cleanup(func() { config.Get().VarSyntax = previous })

	for _, tc := range []struct {
		syntax, command, want string
	}{
		{"angle", `echo \$host \\$shell_only $host $hostname`, `echo \$host \\$shell_only <host> <hostname>`},
		{"dollar", `echo \$host <host> <hostname>`, `echo \$host $host $hostname`},
		{"both", `echo \$host $host <hostname>`, `echo \$host $host <hostname>`},
	} {
		t.Run(tc.syntax, func(t *testing.T) {
			config.Get().VarSyntax = tc.syntax
			markdown := buildComposeMarkdown("Greeting", "", tc.command)
			if !strings.Contains(markdown, "\n"+tc.want+"\n") {
				t.Errorf("snippet did not preserve the command %q:\n%s", tc.want, markdown)
			}
			if !strings.HasSuffix(markdown, "<!-- cheat\nvar host\nvar hostname\n-->\n") {
				t.Errorf("wrong variable declarations:\n%s", markdown)
			}
		})
	}
}
