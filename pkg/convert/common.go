package convert

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	naviPlaceholderRe = regexp.MustCompile(`<([a-zA-Z0-9_\-\.\/]+)>`)
	tldrPlaceholderRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_\-\.\/]+)\s*\}\}`)
	// shellEnvVarRe matches a shell-style environment variable reference in
	// a navi command body. Navi uses `<name>` for its own placeholders, so any
	// `$NAME` is conventionally a shell env var the user expects to inherit
	// from the calling environment. Captures: `${NAME}` (group 1) and `$NAME`
	// (group 2). cheatmd's linter warns on unknown `$` references, so the
	// converter declares each one as `var NAME := $NAME` to bind the cheatmd
	// var to the env value and silence the warning.
	shellEnvVarRe = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)
)

// varDef holds a navi-style variable definition: an optional shell command and
// optional selector args (the part after `---`).
type varDef struct {
	Shell string
	Args  string
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func filenameBase(filename string) string {
	base := filepath.Base(filename)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func parseCommaList(s string) []string {
	var list []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			list = append(list, part)
		}
	}
	return list
}

// parseNaviVarLine parses `name: shell --- args` and returns (name, def, ok).
func parseNaviVarLine(line string) (string, varDef, bool) {
	name, val, ok := strings.Cut(line, ":")
	if !ok {
		return "", varDef{}, false
	}
	name = strings.TrimSpace(name)
	val = strings.TrimSpace(val)

	if shell, args, ok := strings.Cut(val, "---"); ok {
		return name, varDef{Shell: strings.TrimSpace(shell), Args: strings.TrimSpace(args)}, true
	}
	return name, varDef{Shell: val}, true
}

// rewriteNavi replaces <name> placeholders with $sanitized and returns the
// rewritten command plus the deduped list of original placeholder names.
func rewriteNavi(command string) (string, []string) {
	phs := extractPlaceholders(command, naviPlaceholderRe)
	for _, ph := range phs {
		command = strings.ReplaceAll(command, "<"+ph+">", "$"+sanitizeVarName(ph))
	}
	return command, phs
}

// rewriteTldr replaces {{name}} placeholders with $sanitized.
func rewriteTldr(command string) (string, []string) {
	phs := extractPlaceholders(command, tldrPlaceholderRe)
	for _, ph := range phs {
		sanitized := sanitizeVarName(ph)
		command = strings.ReplaceAll(command, "{{"+ph+"}}", "$"+sanitized)
		command = strings.ReplaceAll(command, "{{ "+ph+" }}", "$"+sanitized)
		command = strings.ReplaceAll(command, "{{"+ph+" }}", "$"+sanitized)
		command = strings.ReplaceAll(command, "{{ "+ph+"}}", "$"+sanitized)
	}
	return command, phs
}

// rewriteBoth applies navi then tldr rewriting and returns the union of
// placeholders in encounter order. Used by the cheat/cheat format, which mixes
// both placeholder styles.
func rewriteBoth(command string) (string, []string) {
	command, navi := rewriteNavi(command)
	command, tldr := rewriteTldr(command)
	seen := make(map[string]struct{}, len(navi)+len(tldr))
	all := make([]string, 0, len(navi)+len(tldr))
	for _, ph := range append(navi, tldr...) {
		if _, ok := seen[ph]; ok {
			continue
		}
		seen[ph] = struct{}{}
		all = append(all, ph)
	}
	return command, all
}

func extractPlaceholders(cmd string, re *regexp.Regexp) []string {
	matches := re.FindAllStringSubmatch(cmd, -1)
	var list []string
	seen := make(map[string]struct{})
	for _, m := range matches {
		ph := strings.TrimSpace(m[1])
		if _, ok := seen[ph]; !ok {
			seen[ph] = struct{}{}
			list = append(list, ph)
		}
	}
	return list
}

func sanitizeVarName(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteByte(c)
		} else {
			sb.WriteByte('_')
		}
	}
	res := sb.String()
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	res = strings.Trim(res, "_")
	if res == "" {
		return "v"
	}
	if res[0] >= '0' && res[0] <= '9' {
		res = "v_" + res
	}
	return res
}

func consumeCodeBlock(lines []string, index int) (string, int) {
	var cb strings.Builder
	for index < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[index]), "```") {
		cb.WriteString(lines[index])
		cb.WriteByte('\n')
		index++
	}
	return strings.TrimSpace(cb.String()), index
}

// formatHeaderVarsBlock emits a `<!-- cheat ... -->` block declaring each
// placeholder as a bare selector with the original name as its header label.
// Used by tldr and cheat outputs, where there's no upstream var definition.
func formatHeaderVarsBlock(placeholders []string) string {
	if len(placeholders) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<!-- cheat\n")
	for _, ph := range placeholders {
		sb.WriteString("var ")
		sb.WriteString(sanitizeVarName(ph))
		sb.WriteString(" = --- --header ")
		sb.WriteString(strconv.Quote(ph))
		sb.WriteByte('\n')
	}
	sb.WriteString("-->\n")
	return sb.String()
}

// parseFrontMatterTags extracts `tags:` from a leading `---` YAML block and
// returns the tags plus the index of the first line after the block. Returns
// (nil, 0) when there is no front matter.
func parseFrontMatterTags(lines []string) ([]string, int) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, 0
	}
	var fm []string
	i := 1
	for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
		fm = append(fm, lines[i])
		i++
	}
	if i >= len(lines) {
		return nil, 0
	}
	return parseCheatYAMLTags(fm), i + 1
}

func parseCheatYAMLTags(fm []string) []string {
	for _, line := range fm {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= 5 && strings.EqualFold(trimmed[:5], "tags:") {
			return parseTagsLine(strings.TrimSpace(trimmed[5:]))
		}
	}
	return nil
}

func parseTagsLine(line string) []string {
	line = strings.TrimPrefix(line, "[")
	line = strings.TrimSuffix(line, "]")
	var tags []string
	for _, t := range strings.Split(line, ",") {
		if t = strings.Trim(strings.TrimSpace(t), "\"'"); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// writeHashtags writes each tag as `#tag ` followed by a newline. No-op for an
// empty list.
func writeHashtags(sb *strings.Builder, tags []string) {
	if len(tags) == 0 {
		return
	}
	for _, t := range tags {
		sb.WriteByte('#')
		sb.WriteString(strings.ToLower(t))
		sb.WriteByte(' ')
	}
	sb.WriteByte('\n')
}
