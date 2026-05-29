package convert

import (
	"strings"
	"testing"
)

func TestConvertNaviDefinedVarsGoInModule(t *testing.T) {
	// Any var with a `$ def` line belongs in its section's module. Module
	// name is derived from the section's tag list (joined with underscores)
	// so multi-tag sections produce qualified names.
	input := `% git, checkout
# Switch to a branch
git checkout <branch>

$ branch: git branch --format="%(refname:short)" --- --header "Select branch"
`

	converted, err := ConvertNavi(input, "git.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	if !strings.Contains(converted, "## Switch to a branch") {
		t.Errorf("Expected heading, got:\n%s", converted)
	}
	if !strings.Contains(converted, "git checkout $branch") {
		t.Errorf("Expected placeholder replacement, got:\n%s", converted)
	}

	if !strings.Contains(converted, "export git_checkout\n") {
		t.Errorf("Expected `export git_checkout` from tags [git, checkout], got:\n%s", converted)
	}
	if !strings.Contains(converted, "var branch = git branch --format=\"%(refname:short)\" --- --header \"Select branch\"") {
		t.Errorf("Expected var def in module, got:\n%s", converted)
	}

	block := extractCheatBlock(t, converted, "## Switch to a branch")
	if !strings.Contains(block, "import git_checkout\n") {
		t.Errorf("Cheat should import git_checkout, got:\n%s", block)
	}
	if strings.Contains(block, "var branch =") {
		t.Errorf("Cheat should NOT inline branch def (it's in the module), got:\n%s", block)
	}
}

func TestConvertNaviSectionsAreIsolated(t *testing.T) {
	// android.cheat-style file with multiple `% tags` blocks: each section's
	// vars must live in its own module, and cheats from one section must not
	// see another section's vars (unless they `@extends` it).
	input := `% android, emulator
# Start emulator
"$ANDROID_HOME/tools/emulator" -avd <emulator>

$ emulator: emulator -list-avds

% android, Firebase Crashlytics Test
# Enable debug logging
adb -s <device> shell setprop log.tag.CrashlyticsCore DEBUG

$ device: adb devices | grep device | cut -f 1
`

	converted, err := ConvertNavi(input, "android.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	// Each section gets its own export module.
	if !strings.Contains(converted, "export android_emulator\n") {
		t.Errorf("Expected `export android_emulator` module, got:\n%s", converted)
	}
	if !strings.Contains(converted, "export android_firebase_crashlytics_test\n") {
		t.Errorf("Expected `export android_firebase_crashlytics_test` module, got:\n%s", converted)
	}

	// And the modules are NOT merged into a single `export android`.
	if strings.Contains(converted, "export android\n") {
		t.Errorf("Sections must not be merged into a single `export android`, got:\n%s", converted)
	}

	// Emulator cheat imports its own section's module and not the other.
	emulator := extractCheatBlock(t, converted, "## Start emulator")
	if !strings.Contains(emulator, "import android_emulator\n") {
		t.Errorf("Emulator cheat should import android_emulator, got:\n%s", emulator)
	}
	if strings.Contains(emulator, "import android_firebase_crashlytics_test") {
		t.Errorf("Emulator cheat must NOT see firebase section, got:\n%s", emulator)
	}

	// Firebase cheat imports its own section's module and not the other.
	fb := extractCheatBlock(t, converted, "## Enable debug logging")
	if !strings.Contains(fb, "import android_firebase_crashlytics_test\n") {
		t.Errorf("Firebase cheat should import android_firebase_crashlytics_test, got:\n%s", fb)
	}
	if strings.Contains(fb, "import android_emulator") {
		t.Errorf("Firebase cheat must NOT see emulator section, got:\n%s", fb)
	}

	// And each module contains only its section's vars.
	emulatorMod := extractModuleBlock(t, converted, "android_emulator")
	if !strings.Contains(emulatorMod, "var emulator =") {
		t.Errorf("emulator module should contain `var emulator =`, got:\n%s", emulatorMod)
	}
	if strings.Contains(emulatorMod, "var device") {
		t.Errorf("emulator module must NOT contain device, got:\n%s", emulatorMod)
	}

	fbMod := extractModuleBlock(t, converted, "android_firebase_crashlytics_test")
	if !strings.Contains(fbMod, "var device =") {
		t.Errorf("firebase module should contain `var device =`, got:\n%s", fbMod)
	}
	if strings.Contains(fbMod, "var emulator") {
		t.Errorf("firebase module must NOT contain emulator, got:\n%s", fbMod)
	}
}

func TestConvertNaviExtendsReachesAllMatchingSections(t *testing.T) {
	// A cheat with `@android` reaches every section (across any file) that
	// declares the `android` tag, including sibling sections in the same file.
	input := `% android, emulator
$ emulator: emulator -list-avds

% android, Firebase Crashlytics Test
$ device: adb devices | grep device | cut -f 1

% other
# uses both via @android
@ android
echo <emulator> <device>
`

	converted, err := ConvertNavi(input, "android.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	block := extractCheatBlock(t, converted, "## uses both via @android")
	if !strings.Contains(block, "import android_emulator\n") {
		t.Errorf("Cross-section cheat should import android_emulator, got:\n%s", block)
	}
	if !strings.Contains(block, "import android_firebase_crashlytics_test\n") {
		t.Errorf("Cross-section cheat should import android_firebase_crashlytics_test, got:\n%s", block)
	}
}

func TestConvertNaviBareVarsStayInline(t *testing.T) {
	// A var referenced via `<x>` with no `$ x: ...` definition must be
	// declared inline under the cheat as `var x`, never exported.
	input := `% solo
# uses url and bar; url has a def, bar does not
echo <url> <bar>

$ url: cat urls.txt
`

	converted, err := ConvertNavi(input, "solo.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	// url has a def → goes in module
	if !strings.Contains(converted, "export solo") {
		t.Errorf("Expected export module (url has a def), got:\n%s", converted)
	}
	if !strings.Contains(converted, "var url = cat urls.txt") {
		t.Errorf("Expected `var url = cat urls.txt` in module, got:\n%s", converted)
	}

	block := extractCheatBlock(t, converted, "## uses url and bar")
	if !strings.Contains(block, "import solo") {
		t.Errorf("Cheat should import solo for url, got:\n%s", block)
	}
	// bar has no def → inline bare under the cheat
	if !strings.Contains(block, "var bar\n") {
		t.Errorf("Expected bare `var bar` inline (no def), got:\n%s", block)
	}
	// bar must NOT be in the module
	if strings.Contains(extractModuleBlock(t, converted, "solo"), "var bar") {
		t.Errorf("Bare var bar must not appear in export module, got:\n%s", converted)
	}
}

func TestConvertNaviCheatOnlyImportsModulesItUses(t *testing.T) {
	// A cheat with no defined-var references must NOT import the file's
	// module, even if other cheats in the file do.
	input := `% mixed
# uses defined var
echo <x>

# uses only a bare var
echo <y>

$ x: ls
`

	converted, err := ConvertNavi(input, "mixed.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	using := extractCheatBlock(t, converted, "## uses defined var")
	if !strings.Contains(using, "import mixed") {
		t.Errorf("Defined-var cheat should import mixed, got:\n%s", using)
	}

	notUsing := extractCheatBlock(t, converted, "## uses only a bare var")
	if strings.Contains(notUsing, "import mixed") {
		t.Errorf("Bare-only cheat must NOT import mixed, got:\n%s", notUsing)
	}
	if !strings.Contains(notUsing, "var y\n") {
		t.Errorf("Bare-only cheat should inline var y, got:\n%s", notUsing)
	}
}

func TestConvertNaviBindsShellEnvVars(t *testing.T) {
	// A `$NAME` reference in a navi command body is a shell env var (navi
	// itself uses `<name>` for its placeholders). The converter must declare
	// each such reference as `var NAME := $NAME` so cheatmd's linter doesn't
	// flag it and so the value comes from the inherited environment.
	input := `% deploy
# deploy to env
deploy --target=$DEPLOY_ENV --token=${API_TOKEN} <region>

# uses curly form only
echo ${HOME}/bin
`

	converted, err := ConvertNavi(input, "deploy.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	first := extractCheatBlock(t, converted, "## deploy to env")
	for _, want := range []string{
		"var DEPLOY_ENV := $DEPLOY_ENV",
		"var API_TOKEN := $API_TOKEN",
		"var region\n",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("Expected %q in first cheat block, got:\n%s", want, first)
		}
	}

	second := extractCheatBlock(t, converted, "## uses curly form only")
	if !strings.Contains(second, "var HOME := $HOME") {
		t.Errorf("Expected `var HOME := $HOME` from ${HOME}, got:\n%s", second)
	}
}

func TestConvertNaviShellVarNameCollisionWithPlaceholder(t *testing.T) {
	// If the navi command has both `<NAME>` and `$NAME` (same name), the
	// placeholder wins and the shell-var declaration is skipped to avoid
	// duplicate `var NAME` lines in the same block.
	input := `% t
# collide
echo <NAME> $NAME
`

	converted, err := ConvertNavi(input, "t.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}
	block := extractCheatBlock(t, converted, "## collide")
	if !strings.Contains(block, "var NAME\n") {
		t.Errorf("Expected bare `var NAME` from placeholder, got:\n%s", block)
	}
	if strings.Contains(block, "var NAME := $NAME") {
		t.Errorf("Did not expect `var NAME := $NAME` when placeholder occupies the name, got:\n%s", block)
	}
}

func TestConvertNaviExtendsAcrossFiles(t *testing.T) {
	// A cheat with @extends should reach into sibling files for var defs.
	// cheats only import sibling modules when the cheat actually uses one
	// of that sibling's defined vars.
	utils := NaviSource{
		Path: "git_utils.cheat",
		Content: `% git_utils
# helper
echo helper

$ branch: git branch --format="%(refname:short)"
$ remote: git remote
`,
	}
	main := NaviSource{
		Path: "git.cheat",
		Content: `% git
# checkout
@ git_utils
git checkout <branch>

# fetch
@ git_utils
git fetch <remote>

# unrelated, no extends
echo <something>
`,
	}

	results := ConvertNaviTree([]NaviSource{utils, main})
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	mainOut := findResult(t, results, "git.cheat")
	co := extractCheatBlock(t, mainOut, "## checkout")
	if !strings.Contains(co, "import git_utils") {
		t.Errorf("checkout cheat should import git_utils for branch, got:\n%s", co)
	}
	if strings.Contains(co, "var branch") {
		t.Errorf("checkout cheat should NOT inline branch (resolved via @extends), got:\n%s", co)
	}

	fetch := extractCheatBlock(t, mainOut, "## fetch")
	if !strings.Contains(fetch, "import git_utils") {
		t.Errorf("fetch should import git_utils for remote, got:\n%s", fetch)
	}

	unrelated := extractCheatBlock(t, mainOut, "## unrelated, no extends")
	if strings.Contains(unrelated, "import git_utils") {
		t.Errorf("Cheat without @extends must NOT import git_utils, got:\n%s", unrelated)
	}
	if !strings.Contains(unrelated, "var something\n") {
		t.Errorf("Cheat with no def for something should inline it, got:\n%s", unrelated)
	}

	// git.cheat has no `$` defs of its own → no export module
	if strings.Contains(mainOut, "export git\n") {
		t.Errorf("git.cheat has no own defs, should not emit export git, got:\n%s", mainOut)
	}

	// git_utils.cheat exports both branch and remote
	utilsOut := findResult(t, results, "git_utils.cheat")
	if !strings.Contains(utilsOut, "export git_utils") {
		t.Errorf("git_utils.cheat should emit export git_utils, got:\n%s", utilsOut)
	}
	if !strings.Contains(utilsOut, "var branch = git branch") {
		t.Errorf("Expected branch def in git_utils module, got:\n%s", utilsOut)
	}
	if !strings.Contains(utilsOut, "var remote = git remote") {
		t.Errorf("Expected remote def in git_utils module, got:\n%s", utilsOut)
	}
}

func findResult(t *testing.T, results []NaviResult, path string) string {
	t.Helper()
	for _, r := range results {
		if r.Path == path {
			return r.Content
		}
	}
	t.Fatalf("result for %q not found", path)
	return ""
}

// extractModuleBlock returns the content of the per-file export module for
// the given module name. Used to assert what does and doesn't end up exported.
func extractModuleBlock(t *testing.T, converted, moduleName string) string {
	t.Helper()
	marker := "<!-- cheat\nexport " + moduleName + "\n"
	start := strings.Index(converted, marker)
	if start == -1 {
		return ""
	}
	rest := converted[start:]
	end := strings.Index(rest, "-->")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// extractCheatBlock returns the substring from `heading` up to the next
// section boundary: either the next `## ` heading or the start of a shared
// `<!-- cheat\nexport ` module (which trails the final cheat). EOF is the
// last resort.
func extractCheatBlock(t *testing.T, converted, heading string) string {
	t.Helper()
	start := strings.Index(converted, heading)
	if start == -1 {
		t.Fatalf("heading %q not found in:\n%s", heading, converted)
	}
	rest := converted[start+len(heading):]
	end := earliestIndex(rest, "\n## ", "\n<!-- cheat\nexport ")
	if end == -1 {
		return converted[start:]
	}
	return converted[start : start+len(heading)+end]
}

// earliestIndex returns the smallest non-negative result of strings.Index
// across `needles`, or -1 if none match.
func earliestIndex(s string, needles ...string) int {
	best := -1
	for _, n := range needles {
		i := strings.Index(s, n)
		if i == -1 {
			continue
		}
		if best == -1 || i < best {
			best = i
		}
	}
	return best
}

func TestConvertNaviSelectorAdaptation(t *testing.T) {
	input := `% test, selector
# Retrieve items with various options
git log <item>

$ item: git log --oneline --- --header-lines 2 --column 3 --header "Select item"
`

	converted, err := ConvertNavi(input, "test.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	expectedVar := `var item = git log --oneline | tail -n +3 --- --header "Select item" --map "cut -f3"`
	if !strings.Contains(converted, expectedVar) {
		t.Errorf("Expected var definition:\n%s\ngot:\n%s", expectedVar, converted)
	}
}

func TestConvertNaviAcceptsHeadersAlias(t *testing.T) {
	// Real-world navi cheats sometimes use --headers in place of the
	// spec-documented --header-lines. We accept both.
	input := `% t
# pick
echo <x>

$ x: cmd --- --headers 2
`
	converted, err := ConvertNavi(input, "t.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}
	if !strings.Contains(converted, "| tail -n +3") {
		t.Errorf("Expected --headers alias to trigger tail rewrite, got:\n%s", converted)
	}
}

func TestConvertNaviDropsFzfOnlyFlags(t *testing.T) {
	// fzf-only flags should not appear in the cheatmd output.
	input := `% t
# pick
echo <x>

$ x: cmd --- --query foo --filter bar --preview "cat" --preview-window right --fzf-overrides "--cycle" --prevent-extra --expand --header "keep me" --multi
`
	converted, err := ConvertNavi(input, "t.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}
	for _, dropped := range []string{"--query", "--filter", "--preview", "--preview-window", "--fzf-overrides", "--prevent-extra", "--expand"} {
		if strings.Contains(converted, dropped) {
			t.Errorf("Expected %q to be dropped, got:\n%s", dropped, converted)
		}
	}
	// --header and --multi are cheatmd-supported and should survive.
	for _, kept := range []string{`--header "keep me"`, "--multi"} {
		if !strings.Contains(converted, kept) {
			t.Errorf("Expected %q to be preserved, got:\n%s", kept, converted)
		}
	}
}

func TestConvertNaviColumnWithTabDelimiter(t *testing.T) {
	input := `% rsa
# Generate RSA key
openssl genrsa -out key.pem <length>

$ length: printf "KEY LENGTH\tCOMMENT\n2048\t\tDefault\n4096\t\tBetter" | tail -n +2 --- --column 1 --delimiter "\t"
`

	converted, err := ConvertNavi(input, "rsa.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	// Tab is cut's default delimiter, so no -d should be emitted.
	expected := `--map "cut -f1"`
	if !strings.Contains(converted, expected) {
		t.Errorf("Expected %q in output, got:\n%s", expected, converted)
	}
	if strings.Contains(converted, "--select-column") {
		t.Errorf("Did not expect --select-column in output, got:\n%s", converted)
	}
}

func TestConvertNaviColumnWithCommaDelimiter(t *testing.T) {
	input := `% csv
# Pick a row
echo got <field>

$ field: cat data.csv --- --column 2 --delimiter ","
`

	converted, err := ConvertNavi(input, "csv.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	expected := `--map "cut -f2 -d ','"`
	if !strings.Contains(converted, expected) {
		t.Errorf("Expected %q in output, got:\n%s", expected, converted)
	}
}

func TestConvertNaviColumnChainsExistingMap(t *testing.T) {
	input := `% chain
# Pick a thing
echo <x>

$ x: cmd --- --column 1 --map "tr A-Z a-z"
`

	converted, err := ConvertNavi(input, "chain.cheat")
	if err != nil {
		t.Fatalf("ConvertNavi failed: %v", err)
	}

	expected := `--map "cut -f1 | tr A-Z a-z"`
	if !strings.Contains(converted, expected) {
		t.Errorf("Expected %q in output, got:\n%s", expected, converted)
	}
}

func TestConvertTldr(t *testing.T) {
	input := `# tar

> Archiving utility.
> More information: <https://www.gnu.org/software/tar/>.

- Create a gzipped archive:
` + "```" + `sh
tar -czvf {{path/to/archive.tar.gz}} {{path/to/directory}}
` + "```" + `
`

	converted, err := ConvertTldr(input, "tar.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "## Create a gzipped archive") {
		t.Errorf("Expected heading '## Create a gzipped archive', got: %s", converted)
	}

	if !strings.Contains(converted, "tar -czvf $path_to_archive_tar_gz $path_to_directory") {
		t.Errorf("Expected command placeholders to be replaced, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_archive_tar_gz = --- --header \"path/to/archive.tar.gz\"") {
		t.Errorf("Expected var definition for first placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for second placeholder, got: %s", converted)
	}
}

func TestConvertCheat(t *testing.T) {
	input := `---
syntax: sh
tags: [ tar, archive ]
---
# To create a gzipped archive:
tar -czvf <archive.tar.gz> {{path/to/directory}}
`

	converted, err := ConvertCheat(input, "tar")
	if err != nil {
		t.Fatalf("ConvertCheat failed: %v", err)
	}

	if !strings.Contains(converted, "## To create a gzipped archive") {
		t.Errorf("Expected heading, got: %s", converted)
	}

	if !strings.Contains(converted, "#tar #archive") {
		t.Errorf("Expected localized hashtags, got: %s", converted)
	}

	if strings.Contains(converted, "Tags: #tar #archive") {
		t.Errorf("Did not expect redundant repeating tags line in body, got: %s", converted)
	}

	if !strings.Contains(converted, "tar -czvf $archive_tar_gz $path_to_directory") {
		t.Errorf("Expected placeholder replacement, got: %s", converted)
	}

	if !strings.Contains(converted, "var archive_tar_gz = --- --header \"archive.tar.gz\"") {
		t.Errorf("Expected var definition for navi placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for tldr placeholder, got: %s", converted)
	}
}
