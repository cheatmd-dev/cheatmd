package convert

import (
	"fmt"
	"strconv"
	"strings"
)

// tldrExample is one parsed `- description: \`command\`` pair from a page.
type tldrExample struct {
	Desc    string
	Command string
}

// tldrPlaceholder captures both the raw `{{...}}` payload and the cheatmd var
// shape it should become. Choices is non-nil when the placeholder represents a
// fixed set (option pair, alternation) so the emitter can produce a picker
// rather than a free-text prompt.
type tldrPlaceholder struct {
	Raw     string
	Name    string
	Choices []string
}

type tldrParser struct {
	commandName string
	currentDesc string
	examples    []tldrExample
}

func (p *tldrParser) parseLine(lines []string, index int) int {
	line := strings.TrimSpace(lines[index])
	if line == "" {
		return index
	}

	if strings.HasPrefix(line, "# ") {
		p.commandName = strings.TrimSpace(line[2:])
		return index
	}
	// `>` lines are header metadata (description, "More information", "See also")
	// per the tldr style guide. They appear above any examples and carry no
	// command content; skip them outright so they don't get mis-parsed.
	if strings.HasPrefix(line, ">") {
		return index
	}
	if strings.HasPrefix(line, "- ") {
		p.currentDesc = strings.TrimSuffix(strings.TrimSpace(line[2:]), ":")
		return index
	}
	if p.currentDesc == "" {
		return index
	}

	command, consumed := readCommand(lines, index)
	if command == "" {
		return consumed
	}
	p.examples = append(p.examples, tldrExample{Desc: p.currentDesc, Command: command})
	p.currentDesc = ""
	return consumed
}

// readCommand returns the command body at lines[index], handling both the
// single-line backtick form `\`cmd\`` (the spec's default) and the fenced
// `\`\`\`sh ... \`\`\`` form (used by some real-world pages).
func readCommand(lines []string, index int) (cmd string, consumed int) {
	line := strings.TrimSpace(lines[index])
	if strings.HasPrefix(line, "`") && strings.HasSuffix(line, "`") && len(line) >= 2 {
		return line[1 : len(line)-1], index
	}
	if strings.HasPrefix(line, "```") {
		body, end := consumeCodeBlock(lines, index+1)
		return body, end
	}
	return "", index
}

// ConvertTldr converts a TLDR markdown cheatsheet to CheatMD format.
func ConvertTldr(content string, filename string) (string, error) {
	lines := strings.Split(content, "\n")
	p := &tldrParser{}
	for i := 0; i < len(lines); i++ {
		i = p.parseLine(lines, i)
	}

	category := capitalize(p.commandName)
	if category == "" {
		category = capitalize(filenameBase(filename))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Converted TLDR for %s\n\n", category)
	for _, ex := range p.examples {
		writeTldrExample(&sb, ex)
	}
	return sb.String(), nil
}

func writeTldrExample(sb *strings.Builder, ex tldrExample) {
	command, placeholders := processTldrCommand(ex.Command)
	fmt.Fprintf(sb, "## %s\n\n", ex.Desc)
	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)
	if block := emitTldrVarBlock(placeholders); block != "" {
		sb.WriteString(block)
	}
	sb.WriteByte('\n')
}

// processTldrCommand walks the command body, classifies each `{{...}}`
// placeholder, assigns unique cheatmd var names (with collision suffixes for
// distinct raw payloads that happen to sanitize to the same identifier), and
// substitutes each occurrence with `$name`. Returns the rewritten command and
// the ordered, deduped placeholder list for var-block emission.
func processTldrCommand(command string) (string, []tldrPlaceholder) {
	raws := extractTldrPlaceholders(command)
	infos := classifyTldrPlaceholders(raws)
	for _, info := range infos {
		command = substituteTldrPlaceholder(command, info.Raw, info.Name)
	}
	return command, infos
}

// substituteTldrPlaceholder replaces every `{{<raw>}}` occurrence (tolerating
// internal whitespace variants from hand-formatted pages) with `$name`.
func substituteTldrPlaceholder(cmd, raw, name string) string {
	dollar := "$" + name
	for _, form := range []string{"{{" + raw + "}}", "{{ " + raw + " }}", "{{ " + raw + "}}", "{{" + raw + " }}"} {
		cmd = strings.ReplaceAll(cmd, form, dollar)
	}
	return cmd
}

// classifyTldrPlaceholders bucket-sorts each raw payload into Simple, Option
// Pair, or Alternation, names it, and resolves any name collisions by
// suffixing with _2, _3, …. Order matches first-appearance so the emitted
// var block reads top-to-bottom in source order.
func classifyTldrPlaceholders(raws []string) []tldrPlaceholder {
	out := make([]tldrPlaceholder, 0, len(raws))
	nameCount := make(map[string]int)
	for _, raw := range raws {
		ph := classifyTldrPlaceholder(raw)
		uniq := nameCount[ph.Name]
		nameCount[ph.Name] = uniq + 1
		if uniq > 0 {
			ph.Name = fmt.Sprintf("%s_%d", ph.Name, uniq+1)
		}
		out = append(out, ph)
	}
	return out
}

func classifyTldrPlaceholder(raw string) tldrPlaceholder {
	if choices, ok := parseOptionPair(raw); ok {
		return tldrPlaceholder{Raw: raw, Choices: choices, Name: nameForOptionPair(choices)}
	}
	if choices, ok := parseAlternation(raw); ok {
		return tldrPlaceholder{Raw: raw, Choices: choices, Name: nameForAlternation(choices)}
	}
	return tldrPlaceholder{Raw: raw, Name: sanitizeVarName(raw)}
}

// parseOptionPair detects `[opt1|opt2]`-wrapped payloads (tldr's "alternate
// short/long flag" form). The full payload must be enclosed in matching
// brackets; bracketed alternation embedded inside a larger payload (e.g.
// `path/to/source.tar[.gz|.bz2|.xz]`) is NOT an option pair and falls through
// to Simple so the user types a literal filename.
func parseOptionPair(raw string) ([]string, bool) {
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, false
	}
	inner := raw[1 : len(raw)-1]
	if !strings.Contains(inner, "|") {
		return nil, false
	}
	parts := splitTrim(inner, "|")
	if len(parts) < 2 {
		return nil, false
	}
	return parts, true
}

// parseAlternation detects bare `a|b|c` payloads. Skips anything with brackets
// or internal whitespace — those shapes are ambiguous (could be globs, paths,
// or commands with flags) and are safer as free-text prompts.
func parseAlternation(raw string) ([]string, bool) {
	if !strings.Contains(raw, "|") {
		return nil, false
	}
	if strings.ContainsAny(raw, "[] ") {
		return nil, false
	}
	parts := splitTrim(raw, "|")
	if len(parts) < 2 {
		return nil, false
	}
	return parts, true
}

// nameForOptionPair derives a cheatmd var name from the longest option,
// stripping leading dashes and suffixing `_flag`. The `_flag` suffix avoids
// the common collision where the same page has both `{{[-m|--message]}}` and
// `{{message}}` placeholders.
func nameForOptionPair(choices []string) string {
	longest := ""
	for _, c := range choices {
		trimmed := strings.TrimLeft(c, "-")
		if len(trimmed) > len(longest) {
			longest = trimmed
		}
	}
	name := sanitizeVarName(longest)
	if name == "" {
		return "flag"
	}
	return name + "_flag"
}

// nameForAlternation uses the first option as the var name basis. Subsequent
// collisions (e.g. two `f|d` placeholders in the same example) are resolved
// by the caller's suffix logic.
func nameForAlternation(choices []string) string {
	if len(choices) == 0 {
		return "v"
	}
	return sanitizeVarName(choices[0])
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// emitTldrVarBlock renders the `<!-- cheat -->` block for an example. Simple
// placeholders become bare prompts; option-pair and alternation placeholders
// become pickers fed by `printf '%s\n' …`, which is portable across echo
// flavors (POSIX echo doesn't always interpret `\n`).
func emitTldrVarBlock(placeholders []tldrPlaceholder) string {
	if len(placeholders) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<!-- cheat\n")
	for _, ph := range placeholders {
		sb.WriteString(emitTldrVarLine(ph))
	}
	sb.WriteString("-->\n")
	return sb.String()
}

func emitTldrVarLine(ph tldrPlaceholder) string {
	header := strconv.Quote(ph.Raw)
	if len(ph.Choices) == 0 {
		return fmt.Sprintf("var %s = printf '%%s\\n' %s --- --header %s\n",
			ph.Name,
			singleQuoteForShell(ph.Raw),
			header,
		)
	}
	args := make([]string, len(ph.Choices))
	for i, c := range ph.Choices {
		args[i] = singleQuoteForShell(c)
	}
	return fmt.Sprintf("var %s = printf '%%s\\n' %s --- --header %s\n",
		ph.Name,
		strings.Join(args, " "),
		header,
	)
}
