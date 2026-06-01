package linter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLint(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErrors  []string
		avoidErrors []string
	}{
		{
			name: "TestLintAcceptsContinuedVarShellPipelines",
			content: `# Network

<!-- cheat
export domain
var domain = printf '%s\n' '$domain' "$(grep -v '^[[:space:]]*#' /etc/hosts \
  | sed -E 's/^[[:space:]]+//; s/[[:space:]]+/ /g' \
  | cut -d' ' -f2- \
  | tr ' ' '\n' \
  | sort -u)" --- --header 'Domains'
-->
`,
			avoidErrors: []string{"unknown DSL keyword \"|\""},
		},
		{
			name: "TestLintAcceptsPromptOnlyVarWithArgs",
			content: `# Deploy

## Sync

` + "```sh" + `
rsync -a $source $dest
` + "```" + `
<!-- cheat
var sync_method = printf 'fast\tFast\nslow\tSlow\n' --- --delimiter '\t'

if $sync_method != slow
var dest --- --header "Destination"
fi

if $sync_method == fast
var dest := /tmp/sync
fi
-->
`,
			avoidErrors: []string{"missing an assignment operator"},
		},
		{
			name: "TestLintReportsInvalidChainLine",
			content: `## Bad

` + "```sh" + `
echo bad
` + "```" + `
<!-- cheat
chain demo nope
-->
`,
			wantErrors: []string{"`chain` step must be a positive number"},
		},
		{
			name: "TestLintReportsChainGaps",
			content: `## Later

` + "```sh" + `
echo later
` + "```" + `
<!-- cheat
chain demo 2
-->
`,
			wantErrors: []string{"chain \"demo\" is missing step 1"},
		},
		{
			name: "TestLintReportsDuplicateCheatNamesAtAnyHeaderLevel",
			content: `# whoami

` + "```sh" + `
whoami
` + "```" + `
<!-- cheat
-->

##### whoami

` + "```sh" + `
id
` + "```" + `
<!-- cheat
-->
`,
			wantErrors: []string{"duplicate cheat name \"whoami\""},
		},
		{
			name: "TestLintAllowsSameHeaderTextWhenOnlyOneIsACheat",
			content: `# Server

<!-- cheat
export interface
var interface
-->

## Server

` + "```sh" + `
python3 -m http.server -b $interface
` + "```" + `
<!-- cheat
import interface
-->
`,
			avoidErrors: []string{"duplicate"},
		},
		{
			name: "TestLintAllowsHeadingWithoutCodeBlock",
			content: `## apt

Alias of [apt-get](#apt_get). All techniques from apt-get apply.

## apt-get

### apt-get shell

` + "```sh" + `
apt-get update
` + "```" + `
<!-- cheat
-->
`,
			avoidErrors: []string{"cheat has no code block"},
		},
		{
			name: "TestLintReportsCheatWithoutH2Header",
			content: `Some intro text with no markdown header.

` + "```sh" + `
whoami
` + "```" + `
<!-- cheat
-->

` + "```sh" + `
id
` + "```" + `
<!-- cheat
-->
`,
			wantErrors: []string{"cheat has no markdown header"},
		},
		{
			name: "TestLintDoesNotWarnUndeclaredVarsWithoutCheatBlock",
			content: `# Scratch

` + "```sh" + `
if [ "$INSTALLED" = 1 ]; then
  echo installed
fi
` + "```" + `
`,
			avoidErrors: []string{"variable \"INSTALLED\" referenced"},
		},
		{
			name: "TestLintAcceptsAnyMarkdownHeaderLevelForCheat",
			content: `#### Deep cheat

` + "```sh" + `
whoami
` + "```" + `
<!-- cheat
-->
`,
			avoidErrors: []string{"cheat has no markdown header"},
		},
		{
			name: "TestLintDoesNotWarnForExportOnlyBlocks",
			content: `# Modules

<!-- cheat
export net_target
var host
var port
-->
`,
			avoidErrors: []string{"has no preceding code block"},
		},
		{
			name: "TestLintSkipsUndeclaredCommandRefsForExportedModules",
			content: `## Shell helper

` + "```sh" + `
echo "$provided_by_consumer"
` + "```" + `
<!-- cheat
export shell_helper
-->
`,
			avoidErrors: []string{"variable \"provided_by_consumer\" referenced"},
		},
		{
			name: "TestLintShellSyntaxDeclarationsAndTemplateRefs",
			content: `## Loop

` + "```sh" + `
for i in {1..10}; do echo <a>.$i; done
` + "```" + `
<!-- cheat
-->
`,
			wantErrors:  []string{"variable \"a\" referenced"},
			avoidErrors: []string{"variable \"i\" referenced"},
		},
		{
			name: "TestLintShellSpecialsDoNotApplyToAngleRefs",
			content: `## Home

` + "```sh" + `
echo "$HOME" "<HOME>" "$1" "${10}"
` + "```" + `
<!-- cheat
-->
`,
			wantErrors: []string{"variable \"HOME\" referenced", "variable \"HOME\" referenced"},
		},
		{
			name: "TestLintPowerShellWarnsForUndeclaredInputButNotAssignment",
			content: `## Parse

` + "```ps1" + `
$obj = ConvertFrom-Json $input_data
` + "```" + `
<!-- cheat
-->
`,
			wantErrors:  []string{"variable \"input_data\" referenced"},
			avoidErrors: []string{"variable \"obj\" referenced"},
		},
		{
			name: "TestLintPowerShellProviderNamespacesDoNotWarn",
			content: `## AppData

` + "```powershell" + `
Get-ChildItem $env:APPDATA\MyApp\
` + "```" + `
<!-- cheat
-->
`,
			avoidErrors: []string{"variable \"env\" referenced"},
		},
		{
			name: "TestLintEmbeddedPowerShellInCmdFence",
			content: `## Cmd PS

` + "```cmd" + `
powershell.exe -c "$e=New-Object -ComObject wscript.shell;$e.Popup('$file_out')"
` + "```" + `
<!-- cheat
var file_out
-->
`,
			avoidErrors: []string{"variable \"e\" referenced"},
		},
		{
			name: "TestLintUnknownLanguageDollarRefsAreStrict",
			content: `## Unknown

` + "```python" + `
print($HOME)
` + "```" + `
<!-- cheat
-->
`,
			wantErrors: []string{"variable \"HOME\" referenced"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			writeFile(t, path, tt.content)

			findings, err := Lint(path)
			if err != nil {
				findings, err = Lint(dir) // fallback to dir if it needs full scan
				if err != nil {
					t.Fatalf("Lint returned error: %v", err)
				}
			}

			for _, want := range tt.wantErrors {
				if !hasFinding(findings, want) {
					t.Errorf("missing finding containing %q\nfindings:\n%s", want, formatFindings(findings))
				}
			}

			for _, avoid := range tt.avoidErrors {
				if hasFinding(findings, avoid) {
					t.Errorf("unexpected finding containing %q\nfindings:\n%s", avoid, formatFindings(findings))
				}
			}
		})
	}
}

func TestLintReportsDSLAndReferenceProblems(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cheats.md")
	writeFile(t, path, `# Cheats

## Broken

`+"```sh"+`
echo "$missing $ok"
`+"```"+`
<!-- cheat
var ok
import nope
wat
if
fi extra
-->
`)

	findings, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	want := []string{
		"import \"nope\" does not resolve",
		"variable \"missing\" referenced",
		"unknown DSL keyword \"wat\"",
		"`if` requires a condition",
		"`fi` takes no arguments",
	}
	for _, msg := range want {
		if !hasFinding(findings, msg) {
			t.Fatalf("missing finding containing %q\nfindings:\n%s", msg, formatFindings(findings))
		}
	}
}

func TestLintReportsDuplicateExportsAndSingleLineSyntax(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.md"), `## Module One

`+"```sh"+`
:
`+"```"+`
<!-- cheat export shared -->
`)
	writeFile(t, filepath.Join(dir, "two.md"), `## Module Two

`+"```sh"+`
:
`+"```"+`
<!-- cheat export shared too-many -->
`)
	writeFile(t, filepath.Join(dir, "three.md"), `## Module Three

`+"```sh"+`
:
`+"```"+`
<!-- cheat export shared -->
`)

	findings, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	if !hasFinding(findings, "`export` name must be a single token") {
		t.Fatalf("missing single-line syntax finding\nfindings:\n%s", formatFindings(findings))
	}
	if !hasFinding(findings, "duplicate export \"shared\"") {
		t.Fatalf("missing duplicate export finding\nfindings:\n%s", formatFindings(findings))
	}
}

func TestLintAcceptsChainAndReportsDuplicateSteps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.md"), `## Step one

`+"```sh"+`
echo one
`+"```"+`
<!-- cheat
chain demo 1
-->
`)
	writeFile(t, filepath.Join(dir, "two.md"), `## Step one duplicate

`+"```sh"+`
echo dup
`+"```"+`
<!-- cheat
chain demo 1
-->
`)

	findings, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	if hasFinding(findings, "unknown DSL keyword \"chain\"") {
		t.Fatalf("chain should be a valid DSL keyword\nfindings:\n%s", formatFindings(findings))
	}
	if !hasFinding(findings, "duplicate chain step \"demo\" 1") {
		t.Fatalf("missing duplicate chain step finding\nfindings:\n%s", formatFindings(findings))
	}
}

func TestLintReportsStructuralWarnings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "structural.md")
	writeFile(t, path, `##

### Repeat

Some notes.

### Repeat

`+"```sh"+`
echo ok
`+"```"+`
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, msg := range []string{"empty markdown header"} {
		if !hasFinding(findings, msg) {
			t.Fatalf("missing structural finding containing %q\nfindings:\n%s", msg, formatFindings(findings))
		}
	}
}

func TestLintPowerShellSyntaxDeclarationsAndAutomatics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ps.md")
	writeFile(t, path, `## Compare

`+"```powershell"+`
while($true) {
  $process = Get-WmiObject Win32_Process
  $process2 = Get-WmiObject Win32_Process
  Compare-Object $process $process2
}
`+"```"+`
<!-- cheat
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"true", "process", "process2"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("PowerShell %s should not warn\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintInfersPowerShellInShellFence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ps_sh.md")
	writeFile(t, path, `## Filter

`+"```sh"+`
Get-Process | Where-Object { $_.Responding -eq $false -or $_.Name -ne $null }
`+"```"+`
<!-- cheat
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"_", "false", "null"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("PowerShell-looking sh fence should not warn for %s\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintEmbeddedTclDeclarationsInShellFence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tcl.md")
	writeFile(t, path, `## Tcl

`+"```sh"+`
tclsh
set s value
gets $s c
set e $c
`+"```"+`
<!-- cheat
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"s", "c", "e"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("embedded Tcl variable %s should not warn\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintEmbeddedPerlAndPHPDeclarations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "embedded.md")
	writeFile(t, path, `## Perl

`+"```sh"+`
perl -e '$s="$server"; my $fh = undef; $content = <$fh>; print $s;'
perl -e 'open(my $handle, ">", "$file_out"); print $handle "ok";'
`+"```"+`
<!-- cheat
var server
var file_out
-->

## PHP

`+"```sh"+`
php -r '$p = array(); $h = proc_open("$cmd", $p, $pipes); echo $pipes[1];'
`+"```"+`
<!-- cheat
var cmd
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"s", "fh", "content", "handle", "p", "h", "pipes"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("embedded interpreter local %s should not warn\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintMethodChainsDoNotWarn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "methods.md")
	writeFile(t, path, `## Methods

`+"```powershell"+`
$obj.Document.Application.ShellExecute("cmd.exe","/c $command","C:\Windows\System32",$null,0)
$com.Application.ActivateMicrosoftApp("5")
`+"```"+`
<!-- cheat
var command
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"obj", "com"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("method chain object %s should not warn\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintShellSingleQuotedRegexDoesNotWarn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grep.md")
	writeFile(t, path, `## Regex

`+"```sh"+`
grep -e '\($_GET\|$REQUEST\)' --color
`+"```"+`
<!-- cheat
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"_GET", "REQUEST"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("single-quoted shell regex %s should not warn\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func TestLintDoesNotTreatHeredocXMLTagsAsAngleRefs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heredoc.md")
	writeFile(t, path, `## XML

`+"```sh"+`
cat >$tmp_file <<EOF
<domain>
  <name>x</name>
  <script path='$cmd_file'/>
</domain>
EOF
`+"```"+`
<!-- cheat
var cmd_file
var tmp_file
-->
`)

	findings, err := Lint(path)
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}

	for _, name := range []string{"domain", "name", "script"} {
		if hasFinding(findings, "variable \""+name+"\" referenced") {
			t.Fatalf("heredoc XML tag %s should not be an angle template ref\nfindings:\n%s", name, formatFindings(findings))
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func hasFinding(findings []Finding, substr string) bool {
	for _, f := range findings {
		if strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func formatFindings(findings []Finding) string {
	var b strings.Builder
	for _, f := range findings {
		b.WriteString(f.Format())
		b.WriteByte('\n')
	}
	return b.String()
}
