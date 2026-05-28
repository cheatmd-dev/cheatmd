package convert

import (
	"fmt"
	"strings"
)

// NaviCheat is a single parsed entry from a navi `.cheat` file.
type NaviCheat struct {
	Description string
	Command     string
	Imports     []string
	Tags        []string
}

type naviParser struct {
	tags    []string
	imports []string
	vars    map[string]varDef
	cheats  []NaviCheat
	active  *NaviCheat
}

func (p *naviParser) parseLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, ";") {
		return
	}

	switch trimmed[0] {
	case '%':
		p.flush()
		p.tags = parseCommaList(trimmed[1:])
		p.imports = nil
		p.vars = make(map[string]varDef)
	case '@':
		exts := parseCommaList(trimmed[1:])
		p.imports = append(p.imports, exts...)
		if p.active != nil {
			p.active.Imports = append(p.active.Imports, exts...)
		}
	case '$':
		if name, def, ok := parseNaviVarLine(trimmed[1:]); ok {
			p.vars[name] = def
		}
	case '#':
		p.flush()
		p.active = &NaviCheat{
			Description: strings.TrimSpace(trimmed[1:]),
			Tags:        p.tags,
			Imports:     p.imports,
		}
	default:
		if p.active == nil {
			return
		}
		if p.active.Command != "" {
			p.active.Command += "\n" + line
		} else {
			p.active.Command = line
		}
	}
}

func (p *naviParser) flush() {
	if p.active != nil {
		p.cheats = append(p.cheats, *p.active)
		p.active = nil
	}
}

// ConvertNavi converts a navi .cheat file to CheatMD format.
func ConvertNavi(content string, filename string) (string, error) {
	p := &naviParser{vars: make(map[string]varDef)}
	for _, line := range strings.Split(content, "\n") {
		p.parseLine(line)
	}
	p.flush()

	return serializeNavi(p.cheats, p.vars, filename), nil
}

func serializeNavi(cheats []NaviCheat, vars map[string]varDef, filename string) string {
	moduleName := strings.ToLower(filenameBase(filename))

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Converted %s Cheatsheet\n\n", capitalize(filenameBase(filename)))
	for _, c := range cheats {
		writeNaviCheat(&sb, c, moduleName)
	}
	writeNaviGlobalVars(&sb, moduleName, vars, cheats)
	return sb.String()
}

func writeNaviCheat(sb *strings.Builder, c NaviCheat, moduleName string) {
	if c.Command == "" {
		return
	}

	command, placeholders := rewriteNavi(c.Command)

	fmt.Fprintf(sb, "## %s\n", c.Description)
	writeHashtags(sb, c.Tags)
	sb.WriteByte('\n')

	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)

	sb.WriteString("<!-- cheat\n")
	if len(placeholders) > 0 {
		fmt.Fprintf(sb, "import %s\n", moduleName)
	}
	for _, imp := range c.Imports {
		fmt.Fprintf(sb, "import %s\n", strings.ToLower(imp))
	}
	sb.WriteString("-->\n\n")
}

func writeNaviGlobalVars(sb *strings.Builder, moduleName string, vars map[string]varDef, cheats []NaviCheat) {
	var usedVars []string
	seen := make(map[string]bool)
	for _, c := range cheats {
		for _, ph := range extractPlaceholders(c.Command, naviPlaceholderRe) {
			if sanitized := sanitizeVarName(ph); !seen[sanitized] {
				seen[sanitized] = true
				usedVars = append(usedVars, ph)
			}
		}
	}

	if len(usedVars) == 0 {
		return
	}

	sb.WriteString("<!-- cheat\n")
	fmt.Fprintf(sb, "export %s\n", moduleName)
	for _, ph := range usedVars {
		writeNaviVarDef(sb, sanitizeVarName(ph), vars[ph])
	}
	sb.WriteString("-->\n")
}

func writeNaviVarDef(sb *strings.Builder, name string, def varDef) {
	shell, args := adaptNaviVar(def.Shell, def.Args)
	switch {
	case args != "" && shell != "":
		fmt.Fprintf(sb, "var %s = %s --- %s\n", name, shell, args)
	case args != "":
		fmt.Fprintf(sb, "var %s --- %s\n", name, args)
	case shell != "":
		fmt.Fprintf(sb, "var %s = %s\n", name, shell)
	default:
		fmt.Fprintf(sb, "var %s\n", name)
	}
}

// adaptNaviVar translates navi-flavored selector args to cheatmd-flavored ones:
//
//   - `--headers N` is consumed and re-encoded as `| tail -n +N+1` on the shell.
//   - `--column N` is renamed to `--select-column N`.
//   - When a column is selected without an explicit `--delimiter`, `\t` is added.
//
// All other tokens pass through in their original order.
func adaptNaviVar(shell, args string) (string, string) {
	if args == "" {
		return shell, args
	}

	tokens := parseSelectorArgs(args)
	var out []string
	var headers int
	var hasColumn, hasDelimiter bool

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		next := ""
		if i+1 < len(tokens) {
			next = tokens[i+1]
		}
		switch tok {
		case "--headers":
			fmt.Sscanf(next, "%d", &headers)
			i++
		case "--column", "--select-column":
			hasColumn = true
			out = append(out, "--select-column", next)
			i++
		case "--delimiter":
			hasDelimiter = true
			out = append(out, "--delimiter", next)
			i++
		default:
			out = append(out, tok)
		}
	}
	if hasColumn && !hasDelimiter {
		out = append(out, "--delimiter", "\\t")
	}

	newShell := strings.TrimSpace(shell)
	if headers > 0 && newShell != "" {
		newShell = fmt.Sprintf("%s | tail -n +%d", newShell, headers+1)
	}
	return newShell, serializeSelectorArgs(out)
}

func parseSelectorArgs(s string) []string {
	var args []string
	var current strings.Builder
	var inQuote bool
	var quoteChar byte

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
			continue
		}
		switch c {
		case '"', '\'':
			inQuote = true
			quoteChar = c
		case ' ', '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func serializeSelectorArgs(args []string) string {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = quoteArg(arg)
	}
	return strings.Join(parts, " ")
}

func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\"'\\") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
