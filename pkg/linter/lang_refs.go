package linter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/parser"
)

type RefKind int

const (
	RefShellParam RefKind = iota
	RefPowerShellParam
	RefAngleTemplate
	RefPerlParam
	refUnknownParam RefKind = 99
)

type Ref struct {
	Name   string
	Kind   RefKind
	Line   int
	Column int
}

func isMissing(ref Ref, declared map[string]bool, cmd string) bool {
	if declared[ref.Name] || declared[strings.ToLower(ref.Name)] {
		return false
	}
	if ref.Kind == RefAngleTemplate {
		return true
	}

	if isShellSpecial(ref.Name) {
		if ref.Kind == RefShellParam {
			return false
		}
	}
	if isPowerShellAutomatic(ref.Name) {
		if ref.Kind == RefPowerShellParam || isLikelyPowerShellCommand(cmd) || strings.Contains(strings.ToLower(cmd), "powershell") || strings.Contains(strings.ToLower(cmd), "pwsh") {
			return false
		}
	}
	if isPerlAutomatic(ref.Name) {
		if ref.Kind == RefPerlParam || strings.Contains(cmd, "perl") {
			return false
		}
	}

	return true
}

func referencedVars(c *parser.Cheat) []Ref {
	var refs []Ref
	seen := make(map[string]bool)

	cmd := c.Command
	kind := dollarRefKind(c.CommandLang)
	lang := lintLanguage(c.CommandLang)

	heredocBodyLines := heredocAngleSuppression(cmd)
	lines := strings.Split(cmd, "\n")
	for i, line := range lines {
		lineNo := c.CommandStart + i
		if c.CommandStart == 0 {
			lineNo = 0
		}
		for col := 0; col < len(line); col++ {
			switch line[col] {
			case '$':
				if lang == "shell" && inSingleQuotedShellText(line, col) {
					continue
				}
				ref, end, ok := scanDollarRef(line, col, kind, lineNo)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%d:%s", ref.Kind, ref.Name)
				if !seen[key] {
					seen[key] = true
					refs = append(refs, ref)
				}
				col = end - 1
			case '<':
				if heredocBodyLines[i] {
					continue
				}
				ref, end, ok := scanAngleRef(line, col, lineNo)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%d:%s", ref.Kind, ref.Name)
				if !seen[key] {
					seen[key] = true
					refs = append(refs, ref)
				}
				col = end
			}
		}
	}
	return refs
}

func scanDollarRef(line string, pos int, kind RefKind, lineNo int) (Ref, int, bool) {
	if pos > 0 && line[pos-1] == '\\' {
		return Ref{}, pos + 1, false
	}
	if pos+1 >= len(line) {
		return Ref{}, pos + 1, false
	}
	if line[pos+1] == '{' {
		j := pos + 2
		if j >= len(line) || !isShellBracedVarChar(line[j], true) {
			return Ref{}, pos + 1, false
		}
		j++
		for j < len(line) && isShellBracedVarChar(line[j], false) {
			j++
		}
		if j >= len(line) || line[j] != '}' {
			return Ref{}, pos + 1, false
		}
		return Ref{Name: line[pos+2 : j], Kind: kind, Line: lineNo, Column: pos + 1}, j + 1, true
	}
	next := line[pos+1]
	if kind == RefShellParam && isShellSingleSpecial(next) {
		return Ref{Name: string(next), Kind: kind, Line: lineNo, Column: pos + 1}, pos + 2, true
	}
	if !parser.IsVarChar(next, true) {
		return Ref{}, pos + 1, false
	}
	j := pos + 2
	for j < len(line) && parser.IsVarChar(line[j], false) {
		j++
	}
	if kind == RefPowerShellParam && j < len(line) && line[j] == ':' {
		return Ref{}, j + 1, false
	}
	return Ref{Name: line[pos+1 : j], Kind: kind, Line: lineNo, Column: pos + 1}, j, true
}

func scanAngleRef(line string, pos int, lineNo int) (Ref, int, bool) {
	j := pos + 1
	if j >= len(line) || !parser.IsVarChar(line[j], true) {
		return Ref{}, pos + 1, false
	}
	j++
	for j < len(line) && parser.IsVarChar(line[j], false) {
		j++
	}
	if j >= len(line) || line[j] != '>' {
		return Ref{}, pos + 1, false
	}
	return Ref{Name: line[pos+1 : j], Kind: RefAngleTemplate, Line: lineNo, Column: pos + 1}, j, true
}

func inSingleQuotedShellText(line string, pos int) bool {
	inSingle := false
	for i := 0; i < pos && i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '\'':
			inSingle = !inSingle
		}
	}
	return inSingle
}

func dollarRefKind(lang string) RefKind {
	switch strings.ToLower(lang) {
	case "", "sh", "bash", "zsh", "fish", "shell":
		return RefShellParam
	case "powershell", "pwsh", "ps1":
		return RefPowerShellParam
	default:
		return refUnknownParam
	}
}

var (
	localDeclRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:for|foreach)[[:space:]]+(?:\()?[[:space:]]*\$?([a-z_][a-z0-9_]*)[[:space:]]+(?:in|w)\b`),
		regexp.MustCompile(`(?i)(?:^|[;&|[:space:]"'])\$?([a-z_][a-z0-9_]*)[[:space:]]*=`),
		regexp.MustCompile(`(?i)\bread(?:[[:space:]]+-[a-z]+)*[[:space:]]+([a-z_][a-z0-9_]*(?:[[:space:]]+[a-z_][a-z0-9_]*)*)`),
		regexp.MustCompile(`(?i)\b(?:local|declare|typeset|export|readonly)[[:space:]]+(?:-[a-z]+[[:space:]]+)*([^;&|]+)`),
		regexp.MustCompile(`(?i)\b(?:set|gets(?:[[:space:]]+\$?[a-z_][a-z0-9_]*)?)[[:space:]]+([a-z_][a-z0-9_]*)\b`),
		regexp.MustCompile(`(?i)\bmy[[:space:]]+\$(?:\{)?([a-z_][a-z0-9_]*)(?:\})?`),
		regexp.MustCompile(`(?i)\b(?:proc_open|exec)\([^;]*,\s*\$([a-z_][a-z0-9_]*)\b`),
		regexp.MustCompile(`(?i)\b(?:param|function[[:space:]]+[a-z_][a-z0-9_-]*)[[:space:]]*\(([^)]*)\)`),
		regexp.MustCompile(`(?i)\$([a-z_][a-z0-9_]*)(?:(?:\.|->)[a-z_][a-z0-9_]*)+\(`),
	}
	psVarInParamRe = regexp.MustCompile(`(?i)\$([a-z_][a-z0-9_]*)`)
)

func addSyntaxDeclarations(cmd string, declared map[string]bool) {
	for _, re := range localDeclRegexes {
		for _, m := range re.FindAllStringSubmatch(cmd, -1) {
			if len(m) < 2 {
				continue
			}
			fullMatchLower := strings.ToLower(m[0])
			if strings.HasPrefix(fullMatchLower, "param") || strings.HasPrefix(fullMatchLower, "function") {
				for _, pm := range psVarInParamRe.FindAllStringSubmatch(m[1], -1) {
					declared[pm[1]] = true
					declared[strings.ToLower(pm[1])] = true
				}
				continue
			}

			if strings.HasPrefix(fullMatchLower, "read") || strings.HasPrefix(fullMatchLower, "local") || strings.HasPrefix(fullMatchLower, "declare") || strings.HasPrefix(fullMatchLower, "typeset") || strings.HasPrefix(fullMatchLower, "export") || strings.HasPrefix(fullMatchLower, "readonly") {
				for _, field := range strings.Fields(m[1]) {
					field = strings.TrimLeft(field, "-")
					if field == "" || strings.Contains(field, "=") {
						field = strings.SplitN(field, "=", 2)[0]
					}
					if isIdentifier(field) {
						declared[field] = true
						declared[strings.ToLower(field)] = true
					}
				}
				continue
			}

			declared[m[1]] = true
			declared[strings.ToLower(m[1])] = true
		}
	}
}

func isLikelyPowerShellCommand(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return false
	}
	first, _ := parser.SplitFirstWord(trimmed)
	firstLower := strings.ToLower(first)
	if strings.Contains(first, "-") {
		verb := strings.SplitN(firstLower, "-", 2)[0]
		switch verb {
		case "add", "clear", "compare", "convert", "copy", "export", "find", "for", "foreach",
			"format", "get", "import", "invoke", "measure", "move", "new", "out", "read",
			"remove", "rename", "resolve", "select", "set", "sort", "start", "stop", "where",
			"write":
			return true
		}
	}
	lower := strings.ToLower(cmd)
	return strings.Contains(cmd, "where-object") ||
		strings.Contains(cmd, "foreach-object") ||
		strings.Contains(lower, "$true") ||
		strings.Contains(lower, "$false") ||
		strings.Contains(lower, "$null")
}

func lintLanguage(lang string) string {
	switch dollarRefKind(lang) {
	case RefShellParam:
		return "shell"
	case RefPowerShellParam:
		return "powershell"
	default:
		return "unknown"
	}
}

func heredocAngleSuppression(cmd string) map[int]bool {
	suppressed := make(map[int]bool)
	lines := strings.Split(cmd, "\n")
	endMarker := ""
	for i, line := range lines {
		if endMarker != "" {
			if strings.TrimSpace(line) == endMarker {
				endMarker = ""
				continue
			}
			suppressed[i] = true
			continue
		}
		if marker, ok := heredocMarker(line); ok {
			endMarker = marker
		}
	}
	return suppressed
}

func heredocMarker(line string) (string, bool) {
	idx := strings.Index(line, "<<")
	if idx == -1 {
		return "", false
	}
	rest := strings.TrimSpace(line[idx+2:])
	if strings.HasPrefix(rest, "-") {
		rest = strings.TrimSpace(rest[1:])
	}
	if rest == "" {
		return "", false
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", false
	}
	marker := strings.Trim(fields[0], `"'`)
	if marker == "" {
		return "", false
	}
	return marker, true
}

func isIdentifier(s string) bool {
	if s == "" || !parser.IsVarChar(s[0], true) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !parser.IsVarChar(s[i], false) {
			return false
		}
	}
	return true
}

func isShellSpecial(name string) bool {
	if name == "" {
		return false
	}
	if allDigits(name) {
		return true
	}
	switch name {
	case "@", "*", "#", "?", "$", "!", "-", "_",
		"RANDOM", "SECONDS", "LINENO", "REPLY", "PPID",
		"PWD", "OLDPWD", "HOME", "USER", "UID", "EUID", "HOSTNAME", "SHELL", "PATH", "IFS":
		return true
	default:
		return false
	}
}

func isPowerShellAutomatic(name string) bool {
	switch strings.ToLower(name) {
	case "true", "false", "null", "_", "psitem", "args", "input", "this",
		"error", "matches", "host", "pid", "pwd", "pshome", "psversiontable":
		return true
	default:
		return false
	}
}

func isPerlAutomatic(name string) bool {
	return name == "_"
}

func isShellBracedVarChar(c byte, first bool) bool {
	if c >= '0' && c <= '9' {
		return true
	}
	return parser.IsVarChar(c, first)
}

func isShellSingleSpecial(c byte) bool {
	return (c >= '0' && c <= '9') || strings.ContainsRune("@*#?$!-_", rune(c))
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// declaredVarNames returns the set of variable names declared for cheat c,
// counting its own `var` lines plus everything reachable through imports.
func declaredVarNames(c *parser.Cheat, index *parser.CheatIndex) map[string]bool {
	declared := make(map[string]bool)

	var walk func(modName string, seen map[string]bool)
	walk = func(modName string, seen map[string]bool) {
		if seen[modName] {
			return
		}
		seen[modName] = true
		mod, ok := index.Modules[modName]
		if !ok {
			return
		}
		for _, v := range mod.Vars {
			declared[v.Name] = true
		}
		for _, sub := range mod.Imports {
			walk(sub, seen)
		}
	}

	seen := make(map[string]bool)
	for _, imp := range c.Imports {
		walk(imp, seen)
	}
	for _, v := range c.Vars {
		declared[v.Name] = true
	}
	return declared
}
