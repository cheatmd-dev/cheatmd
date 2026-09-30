package linter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintModuleImports(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		wantLine      int
	}{
		{
			name:     "standalone missing import",
			content:  "# Module\n<!-- cheat\nexport local\nimport missing\nvar x := hello\n-->\n",
			wantLine: 4,
		},
		{
			name: "standalone valid import",
			content: "# Dependency\n<!-- cheat\nexport dependency\nvar x := hello\n-->\n" +
				"# Module\n<!-- cheat\nexport local\nimport dependency\n-->\n",
		},
		{
			name:     "ordinary cheat missing import",
			content:  "# Command\n```sh\necho hello\n```\n<!-- cheat\nimport missing\n-->\n",
			wantLine: 6,
		},
		{
			name:     "exported cheat missing import reported once",
			content:  "# Command\n```sh\necho hello\n```\n<!-- cheat\nexport local\nimport missing\n-->\n",
			wantLine: 7,
		},
	} {
		for _, mode := range []string{"file", "directory"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "imports.md")
				if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
					t.Fatal(err)
				}
				target := path
				if mode == "directory" {
					target = dir
				}
				findings, err := Lint(target)
				if err != nil {
					t.Fatal(err)
				}
				if tc.wantLine == 0 {
					if len(findings) != 0 {
						t.Fatalf("valid import produced findings: %+v", findings)
					}
					return
				}
				want := Finding{File: path, Line: tc.wantLine, Column: 8, Severity: SeverityError, Message: `import "missing" does not resolve to any exported module`}
				if len(findings) != 1 || findings[0] != want {
					t.Fatalf("Lint() = %+v, want [%+v]", findings, want)
				}
			})
		}
	}
}
