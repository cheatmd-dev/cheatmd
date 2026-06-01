package convert

import (
	"fmt"
	"sort"
	"strings"
)

// NaviCheat is a single parsed entry from a navi `.cheat` file.
// Imports holds the section-level `@ extends` tags inherited at parse time
// plus any cheat-level `@` lines added after the heading.
type NaviCheat struct {
	Description string
	Command     string
	Tags        []string // tags of the section this cheat lives in
	Imports     []string
}

// NaviSection is a `% tags` scope: every cheat declared until the next `%`
// shares this section's Vars and contributes to its Module's export. Sections
// are the granularity at which `@extends` resolves — a cheat with `@tag`
// reaches sections (across any file) whose Tags list contains `tag`.
type NaviSection struct {
	Tags     []string
	Imports  []string // section-level @extends (inherited by all cheats here)
	Vars     map[string]varDef
	VarOrder []string // declaration order, for deterministic module output
	Cheats   []NaviCheat
	// Module is the section's export name. Computed from the section's tag
	// list so multiple `% tag1, tag2` blocks in the same file produce
	// distinct module names instead of collapsing into a single file-wide
	// module.
	Module string
}

// NaviFile is one parsed navi cheat file: an ordered list of sections.
type NaviFile struct {
	Filename string
	Sections []*NaviSection
}

// NaviIndex resolves cross-file `@extends` references at section granularity.
// A single tag may appear under sections in many files, and `@tag` reaches
// every one of them.
type NaviIndex struct {
	Files []*NaviFile
	ByTag map[string][]*NaviSection
}

// NaviSource/NaviResult are the directory-mode I/O pair.
type NaviSource struct {
	Path    string
	Content string
}

// NaviResult represents the intermediate parsing state of a single .navi file
// before it is serialized into the final cheatmd Markdown AST.
type NaviResult struct {
	Path    string
	Content string
}

// ----------------------------------------------------------------------------
// Parser
// ----------------------------------------------------------------------------

type naviParser struct {
	sections []*NaviSection
	current  *NaviSection
	active   *NaviCheat
}

func newNaviParser(filename string) *naviParser {
	// Seed with an untagged section so any `# desc` lines that appear before
	// the first `%` still attach somewhere. The fallback module name uses the
	// file base, mirroring navi's behaviour where leading lines belong to the
	// file's implicit topic.
	initial := &NaviSection{
		Vars:   make(map[string]varDef),
		Module: strings.ToLower(filenameBase(filename)),
	}
	return &naviParser{
		sections: []*NaviSection{initial},
		current:  initial,
	}
}

func (p *naviParser) parseLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, ";") {
		return
	}

	switch trimmed[0] {
	case '%':
		p.flushCheat()
		p.startSection(parseCommaList(trimmed[1:]))
	case '@':
		exts := parseCommaList(trimmed[1:])
		p.current.Imports = append(p.current.Imports, exts...)
		if p.active != nil {
			p.active.Imports = append(p.active.Imports, exts...)
		}
	case '$':
		if name, def, ok := parseNaviVarLine(trimmed[1:]); ok {
			if _, exists := p.current.Vars[name]; !exists {
				p.current.VarOrder = append(p.current.VarOrder, name)
			}
			p.current.Vars[name] = def
		}
	case '#':
		p.flushCheat()
		p.active = &NaviCheat{
			Description: strings.TrimSpace(trimmed[1:]),
			Tags:        p.current.Tags,
			Imports:     append([]string(nil), p.current.Imports...),
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

func (p *naviParser) startSection(tags []string) {
	s := &NaviSection{
		Tags:   tags,
		Vars:   make(map[string]varDef),
		Module: computeModuleName(tags),
	}
	p.sections = append(p.sections, s)
	p.current = s
}

func (p *naviParser) flushCheat() {
	if p.active != nil {
		p.current.Cheats = append(p.current.Cheats, *p.active)
		p.active = nil
	}
}

// ParseNaviFile parses a navi file into a NaviFile with per-section structure.
// Empty leading sections (no cheats, no vars) are dropped so they don't show
// up in the index as anonymous noise.
func ParseNaviFile(content, filename string) *NaviFile {
	p := newNaviParser(filename)
	for _, line := range strings.Split(content, "\n") {
		p.parseLine(line)
	}
	p.flushCheat()

	file := &NaviFile{Filename: filename}
	for _, s := range p.sections {
		if len(s.Cheats) == 0 && len(s.VarOrder) == 0 {
			continue
		}
		file.Sections = append(file.Sections, s)
	}
	return file
}

// computeModuleName turns a section's `% tags` list into a single
// underscore-joined identifier, sanitized so it's a valid cheatmd var-name
// shape. Two sections in the same file with different tag lists therefore
// produce distinct module names.
func computeModuleName(tags []string) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		if name := sanitizeVarName(t); name != "" {
			parts = append(parts, strings.ToLower(name))
		}
	}
	return strings.Join(parts, "_")
}

// ----------------------------------------------------------------------------
// Index
// ----------------------------------------------------------------------------

// BuildNaviIndex registers every section under each of its tags so a cheat's
// `@extends` list can fan out to every matching section across the tree.
func BuildNaviIndex(files []*NaviFile) *NaviIndex {
	idx := &NaviIndex{
		Files: files,
		ByTag: make(map[string][]*NaviSection),
	}
	for _, f := range files {
		indexNaviFile(idx, f)
	}
	return idx
}

func indexNaviFile(idx *NaviIndex, f *NaviFile) {
	for _, s := range f.Sections {
		indexNaviSection(idx, s)
	}
}

func indexNaviSection(idx *NaviIndex, s *NaviSection) {
	for _, tag := range s.Tags {
		idx.ByTag[tag] = append(idx.ByTag[tag], s)
	}
}

// ----------------------------------------------------------------------------
// Entry points
// ----------------------------------------------------------------------------

// ConvertNavi converts a single navi .cheat file with no cross-file context.
// `@extends` references that can't be resolved within this file fall back to
// inline bare vars.
func ConvertNavi(content string, filename string) (string, error) {
	file := ParseNaviFile(content, filename)
	idx := BuildNaviIndex([]*NaviFile{file})
	return SerializeNaviFile(file, idx), nil
}

// ConvertNaviTree converts every navi cheat file in a tree using a shared
// index, so `@extends` references can resolve var defs in sibling files.
func ConvertNaviTree(sources []NaviSource) []NaviResult {
	files := make([]*NaviFile, len(sources))
	for i, src := range sources {
		files[i] = ParseNaviFile(src.Content, src.Path)
	}
	idx := BuildNaviIndex(files)

	results := make([]NaviResult, len(files))
	for i, f := range files {
		results[i] = NaviResult{
			Path:    f.Filename,
			Content: SerializeNaviFile(f, idx),
		}
	}
	return results
}

// ----------------------------------------------------------------------------
// Serialization
// ----------------------------------------------------------------------------

// SerializeNaviFile renders a file as cheatmd markdown. Cheats are emitted in
// section order, and each section's export module follows its cheats so the
// reader can see definitions next to the cheats that use them.
func SerializeNaviFile(file *NaviFile, idx *NaviIndex) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Converted %s Cheatsheet\n\n", capitalize(filenameBase(file.Filename)))
	for _, section := range file.Sections {
		writeNaviSectionCheats(&sb, section, idx)
		if len(section.VarOrder) > 0 {
			writeSectionModule(&sb, section)
		}
	}
	return sb.String()
}

func writeNaviSectionCheats(sb *strings.Builder, section *NaviSection, idx *NaviIndex) {
	for _, c := range section.Cheats {
		writeNaviCheat(sb, c, section, idx)
	}
}

// writeNaviCheat emits one cheat block. For each placeholder it resolves:
//   - to its own section's module (if defined here),
//   - to a sibling section's module reached through an `@extends` tag,
//   - or to an inline bare `var name` when no `$` definition is reachable.
//
// Shell env var references (`$NAME` / `${NAME}` in the original navi command)
// get bound as `var NAME := $NAME` literals so cheatmd's linter doesn't flag
// them and the value comes from the runtime environment.
//
// A cheat imports a section's module only when it actually uses one of that
// section's defined vars, so a `@extends` tag never drags unused vars in.
func writeNaviCheat(sb *strings.Builder, c NaviCheat, section *NaviSection, idx *NaviIndex) {
	if c.Command == "" {
		return
	}

	shellEnvVars := extractShellEnvVars(c.Command, placeholderSet(c.Command))
	command, placeholders := rewriteNavi(c.Command)

	fmt.Fprintf(sb, "## %s\n", c.Description)
	writeHashtags(sb, c.Tags)
	sb.WriteByte('\n')
	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)

	imports, inlineVars := resolvePlaceholders(c, section, idx, placeholders)
	if len(imports) == 0 && len(inlineVars) == 0 && len(shellEnvVars) == 0 {
		sb.WriteByte('\n')
		return
	}

	sb.WriteString("<!-- cheat\n")
	for _, imp := range imports {
		fmt.Fprintf(sb, "import %s\n", imp)
	}
	for _, ph := range inlineVars {
		writeNaviVarDef(sb, sanitizeVarName(ph), varDef{})
	}
	for _, name := range shellEnvVars {
		// `:=` is cheatmd's literal-assignment form; the right-hand side is
		// substituted at runtime, so `$NAME` reads from the inherited env
		// instead of re-prompting.
		fmt.Fprintf(sb, "var %s := $%s\n", name, name)
	}
	sb.WriteString("-->\n\n")
}

// resolvePlaceholders bucket-sorts a cheat's placeholders into the set of
// modules to import and the list of names that need bare inline declarations.
// Imports are alphabetized so the diff is stable across runs.
func resolvePlaceholders(
	c NaviCheat,
	section *NaviSection,
	idx *NaviIndex,
	placeholders []string,
) (imports []string, inline []string) {
	importSet := make(map[string]bool)

	for _, ph := range placeholders {
		if _, ok := section.Vars[ph]; ok {
			importSet[section.Module] = true
			continue
		}
		if mod := resolveExtends(c, section, ph, idx); mod != "" {
			importSet[mod] = true
			continue
		}
		inline = append(inline, ph)
	}

	for m := range importSet {
		imports = append(imports, m)
	}
	sort.Strings(imports)
	return imports, inline
}

// resolveExtends walks the cheat's `@extends` tag list and returns the module
// name of the first foreign section whose Vars map contains placeholder ph.
// The cheat's own section is skipped so the lookup doesn't trivially match
// itself.
func resolveExtends(c NaviCheat, own *NaviSection, ph string, idx *NaviIndex) string {
	for _, ext := range c.Imports {
		if mod := resolveExtendsFromTag(ext, own, ph, idx); mod != "" {
			return mod
		}
	}
	return ""
}

func resolveExtendsFromTag(tag string, own *NaviSection, ph string, idx *NaviIndex) string {
	for _, s := range idx.ByTag[tag] {
		if s == own {
			continue
		}
		if _, ok := s.Vars[ph]; ok {
			return s.Module
		}
	}
	return ""
}

// writeSectionModule emits the section's `export <module>` block listing
// every `$ def` declared under it, in first-seen order.
func writeSectionModule(sb *strings.Builder, section *NaviSection) {
	sb.WriteString("<!-- cheat\n")
	fmt.Fprintf(sb, "export %s\n", section.Module)
	for _, name := range section.VarOrder {
		writeNaviVarDef(sb, sanitizeVarName(name), section.Vars[name])
	}
	sb.WriteString("-->\n\n")
}

// placeholderSet returns the sanitized names of every navi `<varname>` in cmd,
// so extractShellEnvVars can avoid duplicating a declaration for any shell-var
// reference that happens to share its name with a navi placeholder.
func placeholderSet(cmd string) map[string]bool {
	out := make(map[string]bool)
	for _, ph := range extractPlaceholders(cmd, naviPlaceholderRe) {
		out[sanitizeVarName(ph)] = true
	}
	return out
}

// extractShellEnvVars walks cmd for `$NAME` and `${NAME}` references in
// first-seen order, deduped, skipping any name that collides with a navi
// placeholder (which would produce duplicate `var` lines).
func extractShellEnvVars(cmd string, placeholders map[string]bool) []string {
	matches := shellEnvVarRe.FindAllStringSubmatch(cmd, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		if seen[name] || placeholders[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// ----------------------------------------------------------------------------
// Var def emission
// ----------------------------------------------------------------------------

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

// adaptNaviVar translates navi-flavored selector args to cheatmd-flavored ones.
// Reference: https://github.com/denisidoro/navi/blob/master/docs/cheatsheet/syntax/README.md
//
// Mappings:
//   - `--header-lines N` (and the older `--headers` alias) is consumed and
//     re-encoded as `| tail -n +N+1` on the shell. cheatmd doesn't render an
//     fzf header band, so dropping the rows from input matches user intent.
//   - `--column N` (and `--select-column N`) becomes `--map "cut -fN"`. Going
//     through cut bypasses cheatmd's fragile string-split delimiter handling
//     and matches the proven idiomatic cheatmd pattern.
//   - `--delimiter <s>` is consumed (with common escapes decoded) and folded
//     into the cut invocation when non-tab.
//   - `--map "X"` is preserved and chained after cut when both are present.
//   - `--multi`, `--header <s>`, and any other unrecognized tokens are passed
//     through.
//
// Dropped (fzf-only flags that cheatmd has no equivalent for):
//
//	`--prevent-extra`, `--expand`, `--fzf-overrides`, `--query`, `--filter`,
//	`--preview`, `--preview-window`.
func adaptNaviVar(shell, args string) (string, string) {
	if args == "" {
		return shell, args
	}

	tokens := parseSelectorArgs(args)
	var out []string
	var headers int
	var columnN, delimiterVal, userMap string

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		next := ""
		if i+1 < len(tokens) {
			next = tokens[i+1]
		}
		switch tok {
		case "--header-lines", "--headers":
			if _, err := fmt.Sscanf(next, "%d", &headers); err != nil {
				headers = 0
			}
			i++
		case "--column", "--select-column":
			columnN = next
			i++
		case "--delimiter":
			delimiterVal = decodeShellEscapes(next)
			i++
		case "--map":
			userMap = next
			i++
		case "--prevent-extra", "--expand":
			// Valueless fzf-only flags with no cheatmd equivalent. Drop.
		case "--fzf-overrides", "--query", "--filter", "--preview", "--preview-window":
			// fzf-only flags with no cheatmd equivalent. Consume the arg.
			i++
		default:
			out = append(out, tok)
		}
	}

	if mapCmd := buildMapCmd(columnN, delimiterVal, userMap); mapCmd != "" {
		out = append(out, "--map", mapCmd)
	}

	newShell := strings.TrimSpace(shell)
	if headers > 0 && newShell != "" {
		newShell = fmt.Sprintf("%s | tail -n +%d", newShell, headers+1)
	}
	return newShell, serializeSelectorArgs(out)
}

func buildMapCmd(columnN, delimiterVal, userMap string) string {
	if columnN == "" {
		return userMap
	}
	cutPipe := "cut -f" + columnN
	if delimiterVal != "" && delimiterVal != "\t" {
		cutPipe += " -d " + singleQuoteForShell(delimiterVal)
	}
	if userMap != "" {
		return cutPipe + " | " + userMap
	}
	return cutPipe
}

func singleQuoteForShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func decodeShellEscapes(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			sb.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 't':
			sb.WriteByte('\t')
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case '\\':
			sb.WriteByte('\\')
		case '"':
			sb.WriteByte('"')
		default:
			sb.WriteByte(s[i])
			sb.WriteByte(s[i+1])
		}
		i++
	}
	return sb.String()
}

// ----------------------------------------------------------------------------
// Selector arg parsing/serialization
// ----------------------------------------------------------------------------

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
