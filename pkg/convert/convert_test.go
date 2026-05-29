package convert

import (
	"os"
	"path/filepath"
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

	if !strings.Contains(converted, "var path_to_archive_tar_gz = printf '%s\\n' 'path/to/archive.tar.gz' --- --header \"path/to/archive.tar.gz\"") {
		t.Errorf("Expected var definition for first placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = printf '%s\\n' 'path/to/directory' --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for second placeholder, got: %s", converted)
	}
}

// ============================================================================
// TLDR placeholder-shape tests
// ============================================================================
//
// The tldr-pages style guide allows a long tail of placeholder shapes the
// initial converter regex didn't recognize. These tests pin down the shape
// classification, var-name derivation, and command rewriting for each form
// we expect to see in real pages.

func TestConvertTldrOptionPairBecomesPickerWithFlagSuffix(t *testing.T) {
	// `{{[-m|--message]}}` is the tldr "alternate short/long flag" form.
	// It must become a picker fed by `printf '%s\n' '-m' '--message'`, and
	// the var name carries `_flag` to dodge a collision with a sibling
	// `{{message}}` placeholder in the same example (git-commit style).
	input := "# git commit\n\n- Commit with message:\n\n`git commit {{[-m|--message]}} \"{{message}}\"`\n"

	converted, err := ConvertTldr(input, "git-commit.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "git commit $message_flag \"$message\"") {
		t.Errorf("Expected `$message_flag` + `$message`, got:\n%s", converted)
	}
	if !strings.Contains(converted, `var message_flag = printf '%s\n' '-m' '--message' --- --header "[-m|--message]"`) {
		t.Errorf("Expected message_flag picker var, got:\n%s", converted)
	}
	if !strings.Contains(converted, `var message = printf '%s\n' 'message' --- --header "message"`) {
		t.Errorf("Expected free-text message var, got:\n%s", converted)
	}
}

func TestConvertTldrBareAlternationBecomesPicker(t *testing.T) {
	// `{{f|d}}` is a bare alternation (no brackets) — picker over the
	// listed options, named after the first option.
	input := "# find\n\n- Find files or dirs:\n\n`find {{path/to/dir}} -type {{f|d}}`\n"

	converted, err := ConvertTldr(input, "find.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "-type $f") {
		t.Errorf("Expected alternation to substitute as $f, got:\n%s", converted)
	}
	if !strings.Contains(converted, `var f = printf '%s\n' 'f' 'd' --- --header "f|d"`) {
		t.Errorf("Expected alternation picker var, got:\n%s", converted)
	}
}

func TestConvertTldrBracketedAlternationInsidePathIsLiteral(t *testing.T) {
	// `{{path/to/source.tar[.gz|.bz2|.xz]}}` has brackets INSIDE a larger
	// payload. It must NOT be treated as an option pair; the user should
	// type a literal filename whose extension matches the suggested set.
	input := "# tar\n\n- Extract an archive:\n\n`tar xvf {{path/to/source.tar[.gz|.bz2|.xz]}}`\n"

	converted, err := ConvertTldr(input, "tar.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if strings.Contains(converted, "'path/to/source.tar[.gz'") {
		t.Errorf("Bracketed alt inside a path must not be split into picker choices, got:\n%s", converted)
	}
	if !strings.Contains(converted, `var path_to_source_tar_gz_bz2_xz = printf '%s\n' 'path/to/source.tar[.gz|.bz2|.xz]' --- --header "path/to/source.tar[.gz|.bz2|.xz]"`) {
		t.Errorf("Expected the bracketed path to appear as a free-text header label, got:\n%s", converted)
	}
}

func TestConvertTldrGlobAndCommandPlaceholdersSurvive(t *testing.T) {
	// Real-world tldr pages stash globs (`{{*.html}}`), wildcards
	// (`{{*lib*}}`), embedded commands (`{{wc -l}}`), and multi-arg
	// placeholders (`{{path/to/file1 path/to/file2 ...}}`). All must round-
	// trip through the converter and surface as bare prompts whose --header
	// preserves the original payload as a hint.
	input := "# samples\n\n" +
		"- Glob:\n\n`find . -name '{{*.html}}'`\n\n" +
		"- Wildcard:\n\n`find . -iname '{{*lib*}}'`\n\n" +
		"- Embedded command:\n\n`find . -name '{{*.ext}}' -exec {{wc -l}} {} \\;`\n\n" +
		"- Multi-arg:\n\n`tar cf {{path/to/target.tar}} {{path/to/file1 path/to/file2 ...}}`\n"

	converted, err := ConvertTldr(input, "samples.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	// Original `{{...}}` payloads must not survive in the output — every one
	// of them must have been substituted with `$<name>`.
	if strings.Contains(converted, "{{") || strings.Contains(converted, "}}") {
		t.Errorf("All `{{...}}` must be rewritten, got:\n%s", converted)
	}
	for _, header := range []string{
		`"*.html"`,
		`"*lib*"`,
		`"wc -l"`,
		`"path/to/file1 path/to/file2 ..."`,
	} {
		if !strings.Contains(converted, header) {
			t.Errorf("Expected header label %s preserved, got:\n%s", header, converted)
		}
	}
}

func TestConvertTldrJsonPlaceholderWithBracesSurvives(t *testing.T) {
	input := "# curl\n\n- Send JSON:\n\n`curl {{[-d|--data]}} '{{{\"name\":\"bob\"}}}' {{http://example.com/users/1234}}`\n"

	converted, err := ConvertTldr(input, "curl.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "curl $data_flag '$name_bob' $http_example_com_users_1234") {
		t.Errorf("Expected JSON payload to become a single quoted placeholder var, got:\n%s", converted)
	}
	if !strings.Contains(converted, `var name_bob = printf '%s\n' '{"name":"bob"}' --- --header "{\"name\":\"bob\"}"`) {
		t.Errorf("Expected full JSON object preserved as header, got:\n%s", converted)
	}
	if strings.Contains(converted, "{$name_bob}") {
		t.Errorf("JSON braces must not be left around a partial placeholder, got:\n%s", converted)
	}
}

func TestConvertTldrSameSanitizedNameGetsSuffixed(t *testing.T) {
	// Two distinct raw payloads sanitizing to the same identifier must NOT
	// produce duplicate `var X` lines; the second gets `_2`, etc.
	input := "# t\n\n- Two paths:\n\n`cp {{path/to/file}} {{path/to/file.bak}}`\n"

	converted, err := ConvertTldr(input, "t.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}

	if !strings.Contains(converted, "$path_to_file ") {
		t.Errorf("Expected first placeholder as $path_to_file, got:\n%s", converted)
	}
	// Either suffix scheme is OK as long as the second is distinct. Assert
	// non-collision by checking BOTH names are declared.
	if !strings.Contains(converted, "var path_to_file ") {
		t.Errorf("Expected first var declared, got:\n%s", converted)
	}
	if strings.Count(converted, "var path_to_file ") != 1 {
		t.Errorf("First name must appear exactly once (no duplicates), got:\n%s", converted)
	}
}

func TestConvertTldrKeypressSyntaxPassesThrough(t *testing.T) {
	// tldr's `<Enter><~><.>` keypress notation is NOT a placeholder; the
	// converter must let it through verbatim.
	input := "# ssh\n\n- Close session:\n\n`<Enter><~><.>`\n"

	converted, err := ConvertTldr(input, "ssh.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}
	if !strings.Contains(converted, "<Enter><~><.>") {
		t.Errorf("Keypress syntax must survive, got:\n%s", converted)
	}
}

func TestConvertTldrSkipsHeaderInfoLines(t *testing.T) {
	// `>` lines (description, "More information", "See also") are header
	// metadata and must not show up as commands or descriptions.
	input := "# tool\n\n> A description.\n> More information: <https://example.com>.\n> See also: `other`.\n\n- Run it:\n\n`tool {{arg}}`\n"

	converted, err := ConvertTldr(input, "tool.md")
	if err != nil {
		t.Fatalf("ConvertTldr failed: %v", err)
	}
	for _, leak := range []string{"More information", "See also", "https://example.com"} {
		if strings.Contains(converted, leak) {
			t.Errorf("Header line %q leaked into output:\n%s", leak, converted)
		}
	}
}

func TestConvertTldrRealFixturesProduceNoStrayPlaceholders(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "tldr")
	if _, err := os.Stat(fixtureDir); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("real tldr fixture directory %s is not present", fixtureDir)
		}
		t.Fatalf("stat fixture dir %s: %v", fixtureDir, err)
	}

	// Sweep test against committed real-world tldr pages. The contract:
	// every `{{...}}` in the source must be substituted with a `$NAME` in
	// the converted output, and every emitted var line must reference a
	// var that's actually used in the example's command.
	for _, name := range []string{"tar", "curl", "grep", "find", "ssh", "git-commit"} {
		t.Run(name, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", "tldr", name+".md")
			if _, err := os.Stat(fixturePath); err != nil {
				if os.IsNotExist(err) {
					t.Skipf("real tldr fixture %s is not present", fixturePath)
				}
				t.Fatalf("stat fixture %s: %v", fixturePath, err)
			}
			src := readTestdata(t, "tldr/"+name+".md")
			converted, err := ConvertTldr(src, name+".md")
			if err != nil {
				t.Fatalf("ConvertTldr(%s) failed: %v", name, err)
			}
			if strings.Contains(converted, "{{") {
				t.Errorf("Stray `{{` in converted %s, indicating an unrewritten placeholder:\n%s",
					name, converted)
			}
			if strings.Contains(converted, "}}") {
				t.Errorf("Stray `}}` in converted %s:\n%s", name, converted)
			}
			// Round-trip sanity: the number of `## ` headings should match
			// the number of `- ` example bullets in the source.
			wantExamples := strings.Count(src, "\n- ")
			gotExamples := strings.Count(converted, "\n## ")
			if wantExamples != gotExamples {
				t.Errorf("%s: example count drifted: source has %d, converted has %d",
					name, wantExamples, gotExamples)
			}
		})
	}
}

func readTestdata(t *testing.T, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", relPath))
	if err != nil {
		t.Fatalf("read testdata %s: %v", relPath, err)
	}
	return string(data)
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

	if !strings.Contains(converted, "var archive_tar_gz --- --header \"archive.tar.gz\"") {
		t.Errorf("Expected var definition for navi placeholder, got: %s", converted)
	}

	if !strings.Contains(converted, "var path_to_directory = printf '%s\\n' 'path/to/directory' --- --header \"path/to/directory\"") {
		t.Errorf("Expected var definition for tldr placeholder, got: %s", converted)
	}
}

func TestConvertCheatPreservesExampleValuesAsEditableDefaults(t *testing.T) {
	input := `# Curl with header:
curl {{[-H|--header]}} '{{Authorization: Bearer token}}' {{[-X|--request]}} {{GET|POST}} {{https://example.com}}

# JSON body:
curl {{[-d|--data]}} '{{{"name":"bob"}}}' {{http://example.com/users/1234}}
`

	converted, err := ConvertCheat(input, "curl")
	if err != nil {
		t.Fatalf("ConvertCheat failed: %v", err)
	}

	if !strings.Contains(converted, "curl $header_flag '$Authorization_Bearer_token' $request_flag $GET $https_example_com") {
		t.Errorf("Expected curl header command to be rewritten, got:\n%s", converted)
	}
	for _, want := range []string{
		`var header_flag = printf '%s\n' '-H' '--header' --- --header "[-H|--header]"`,
		`var Authorization_Bearer_token = printf '%s\n' 'Authorization: Bearer token' --- --header "Authorization: Bearer token"`,
		`var GET = printf '%s\n' 'GET' 'POST' --- --header "GET|POST"`,
		`var https_example_com = printf '%s\n' 'https://example.com' --- --header "https://example.com"`,
		`var name_bob = printf '%s\n' '{"name":"bob"}' --- --header "{\"name\":\"bob\"}"`,
	} {
		if !strings.Contains(converted, want) {
			t.Errorf("Expected %q in converted cheat, got:\n%s", want, converted)
		}
	}
	if strings.Contains(converted, "{$name_bob}") {
		t.Errorf("JSON placeholder must not be partially rewritten, got:\n%s", converted)
	}
}

func TestConvertCheatMixedPlaceholderNameCollisionsAreUnique(t *testing.T) {
	input := `# Copy:
cp <path/to/file> {{path/to/file.bak}}
`

	converted, err := ConvertCheat(input, "copy")
	if err != nil {
		t.Fatalf("ConvertCheat failed: %v", err)
	}

	if !strings.Contains(converted, "cp $path_to_file $path_to_file_bak") {
		t.Errorf("Expected mixed placeholders to rewrite distinctly, got:\n%s", converted)
	}
	if strings.Count(converted, "var path_to_file ") != 1 {
		t.Errorf("Expected one first var definition, got:\n%s", converted)
	}
	if !strings.Contains(converted, "var path_to_file_bak ") {
		t.Errorf("Expected second var definition, got:\n%s", converted)
	}
}

func TestConvertCheatBracedPlaceholdersBecomePrompts(t *testing.T) {
	input := `# Create a pool:
zpool create ${pool} raidz1 ${device} ${failed-device}

# Preserve escaped shell template:
npm config set //npm.intra/:_authToken=\${NPM_TOKEN}
`

	converted, err := ConvertCheat(input, "zfs")
	if err != nil {
		t.Fatalf("ConvertCheat failed: %v", err)
	}

	if !strings.Contains(converted, "zpool create $pool raidz1 $device $failed_device") {
		t.Errorf("Expected braced placeholders to rewrite, got:\n%s", converted)
	}
	for _, want := range []string{
		`var pool --- --header "pool"`,
		`var device --- --header "device"`,
		`var failed_device --- --header "failed-device"`,
	} {
		if !strings.Contains(converted, want) {
			t.Errorf("Expected %q in converted cheat, got:\n%s", want, converted)
		}
	}
	if !strings.Contains(converted, `\${NPM_TOKEN}`) {
		t.Errorf("Escaped braced shell value should be preserved, got:\n%s", converted)
	}
	if strings.Contains(converted, `var NPM_TOKEN`) {
		t.Errorf("Escaped braced shell value should not become a prompt, got:\n%s", converted)
	}
}
