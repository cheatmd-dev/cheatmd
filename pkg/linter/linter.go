// Package linter checks a cheatmd cheats directory for problems.
//
// It runs three categories of checks:
//
//  1. DSL syntax: each line inside a `<!-- cheat -->` block must be one of
//     `var`/`if`/`fi`/`export`/`import`, blank, or a `#` comment. Var names
//     must be valid; `if` must have a matching `fi`.
//  2. References: every `import foo` must resolve to an `export foo` defined
//     somewhere in the index. Duplicate `export` names are flagged.
//     `$var`/`<var>` references in commands should be declared (in a cheat
//     block or imported module), and undeclared references are warnings.
//  3. Structural: empty `##` headers, missing code blocks, duplicate `##`
//     headers within one file.
package linter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

// Severity classifies a finding.
type Severity int

const (
	// SeverityError indicates a genuine problem: malformed DSL, missing
	// import target, duplicate export, etc.
	SeverityError Severity = iota
	// SeverityWarning indicates a recommendation or potential issue:
	// undeclared variables, duplicate headers within a file.
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	default:
		return "warning"
	}
}

// Finding is one issue surfaced by the linter.
type Finding struct {
	File     string
	Line     int // 1-indexed; 0 means "file-level" (no specific line)
	Column   int // 1-indexed; 0 means "line-level"
	Severity Severity
	Message  string
}

// Format renders the finding in GCC-style:
//
//	file:line:col: severity: message
//
// Line/col are omitted (replaced with 1) when not known.
func (f Finding) Format() string {
	line := f.Line
	if line < 1 {
		line = 1
	}
	col := f.Column
	if col < 1 {
		col = 1
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s", f.File, line, col, f.Severity, f.Message)
}

// Lint walks path (file or directory), parses every markdown file, and
// returns all findings sorted by file/line.
func Lint(path string) ([]Finding, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	p := parser.NewParser()
	var index *parser.CheatIndex
	if info.IsDir() {
		index, err = p.ParseDirectory(path)
	} else {
		index, err = p.ParseSingleFile(path)
	}
	if err != nil {
		return nil, err
	}

	var findings []Finding

	for _, e := range index.Errors {
		severity := SeverityError
		if strings.Contains(e.Message, "cheat has no markdown header") ||
			strings.Contains(e.Message, "block has no preceding code block") {
			severity = SeverityWarning
		}
		findings = append(findings, Finding{
			File:     e.File,
			Line:     e.Line,
			Column:   1,
			Severity: severity,
			Message:  e.Message,
		})
	}

	indexFindings := lintIndex(index, info.IsDir())
	findings = append(findings, indexFindings...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})
	return findings, nil
}

// ============================================================================
// Whole-index checks
// ============================================================================

// lintIndex runs the parser and checks cross-cheat references: missing
// imports, duplicate exports, and undeclared variable references in commands.
func lintIndex(index *parser.CheatIndex, isDir bool) []Finding {
	var findings []Finding

	// Collect duplicate exports
	for _, dup := range index.Duplicates {
		findings = append(findings, Finding{
			File:     dup.File2,
			Line:     1,
			Column:   1,
			Severity: SeverityError,
			Message:  fmt.Sprintf("duplicate export %q (also defined in %s)", dup.Name, filepath.Base(dup.File1)),
		})
	}

	// Check duplicate cheat headers globally
	type cheatLoc struct {
		file string
		line int
	}
	headerLocs := make(map[string]cheatLoc)
	warnedHeaders := make(map[string]bool)
	for _, c := range index.Cheats {
		if c.Header == "" {
			continue
		}
		if firstLoc, exists := headerLocs[c.Header]; exists {
			if !warnedHeaders[c.Header] {
				msg := fmt.Sprintf("duplicate cheat name %q (also `# %s` at line %d)", c.Header, c.Header, firstLoc.line)
				if firstLoc.file != c.File {
					msg = fmt.Sprintf("duplicate cheat name %q (also `# %s` at %s:%d)", c.Header, c.Header, firstLoc.file, firstLoc.line)
				}
				findings = append(findings, Finding{
					File:     c.File,
					Line:     c.HeaderLine,
					Column:   1,
					Severity: SeverityWarning,
					Message:  msg,
				})
				warnedHeaders[c.Header] = true
			}
		} else {
			headerLocs[c.Header] = cheatLoc{file: c.File, line: c.HeaderLine}
		}
	}
	seenChainSteps := make(map[string]*parser.Cheat)
	chainSteps := make(map[string]map[int]*parser.Cheat)

	for _, c := range index.Cheats {
		if c.ChainName != "" {
			if chainSteps[c.ChainName] == nil {
				chainSteps[c.ChainName] = make(map[int]*parser.Cheat)
			}
			chainSteps[c.ChainName][c.ChainStep] = c
			key := fmt.Sprintf("%s:%d", c.ChainName, c.ChainStep)
			if existing := seenChainSteps[key]; existing != nil {
				line, col := findDSLRef(c.File, "chain", c.ChainName)
				findings = append(findings, Finding{
					File:     c.File,
					Line:     line,
					Column:   col,
					Severity: SeverityError,
					Message:  fmt.Sprintf("duplicate chain step %q %d (also in %s)", c.ChainName, c.ChainStep, existing.File),
				})
			} else {
				seenChainSteps[key] = c
			}
		}

		// Missing imports.
		for _, imp := range c.Imports {
			if _, ok := index.Modules[imp]; !ok {
				line, col := findDSLRef(c.File, "import", imp)
				findings = append(findings, Finding{
					File:     c.File,
					Line:     line,
					Column:   col,
					Severity: SeverityError,
					Message:  fmt.Sprintf("import %q does not resolve to any exported module", imp),
				})
			}
		}

		// Undeclared $var / <var> references in the command. Plain code
		// fences without a cheat block are allowed to contain ordinary shell
		// variables; only lint cheats that opted into metadata.
		if c.Export != "" || !c.HasCheatBlock {
			continue
		}
		if len(c.Command) > 0 {
			declared := declaredVarNames(c, index)
			addSyntaxDeclarations(c.Command, declared)
			for _, ref := range referencedVars(c) {
				if isMissing(ref, declared, c.Command) {
					findings = append(findings, Finding{
						File:     c.File,
						Line:     ref.Line,
						Column:   ref.Column,
						Severity: SeverityWarning,
						Message:  fmt.Sprintf("undeclared variable %q referenced in command", ref.Name),
					})
				}
			}
		}
	}
	for name, steps := range chainSteps {
		maxStep := 0
		var first *parser.Cheat
		for step, cheat := range steps {
			if first == nil || step < first.ChainStep {
				first = cheat
			}
			if step > maxStep {
				maxStep = step
			}
		}
		for step := 1; step <= maxStep; step++ {
			if steps[step] != nil {
				continue
			}
			line, col := 0, 0
			file := ""
			if first != nil {
				file = first.File
				line, col = findDSLRef(first.File, "chain", name)
			}
			findings = append(findings, Finding{
				File:     file,
				Line:     line,
				Column:   col,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("chain %q is missing step %d", name, step),
			})
		}
	}
	return findings
}
