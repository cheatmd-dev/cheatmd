package convert

import (
	"fmt"
	"strings"
)

// CheatEntry is a single parsed entry from a cheat/cheat plain-text file.
type CheatEntry struct {
	Desc    string
	Command string
}

type cheatParser struct {
	currentDesc strings.Builder
	currentCmd  strings.Builder
	entries     []CheatEntry
}

func (p *cheatParser) parseLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	if !strings.HasPrefix(trimmed, "#") {
		if p.currentCmd.Len() > 0 {
			p.currentCmd.WriteByte('\n')
		}
		p.currentCmd.WriteString(line)
		return
	}

	p.flush()
	descLine := strings.TrimSpace(trimmed[1:])
	if p.currentDesc.Len() > 0 {
		p.currentDesc.WriteByte(' ')
	}
	p.currentDesc.WriteString(descLine)
}

func (p *cheatParser) flush() {
	if p.currentCmd.Len() == 0 {
		return
	}
	p.entries = append(p.entries, CheatEntry{
		Desc:    strings.TrimSpace(p.currentDesc.String()),
		Command: strings.TrimSpace(p.currentCmd.String()),
	})
	p.currentDesc.Reset()
	p.currentCmd.Reset()
}

// ConvertCheat converts a cheat/cheat plain-text cheatsheet to CheatMD format.
func ConvertCheat(content string, filename string) (string, error) {
	lines := strings.Split(content, "\n")
	tags, startLine := parseFrontMatterTags(lines)

	p := &cheatParser{}
	for i := startLine; i < len(lines); i++ {
		p.parseLine(lines[i])
	}
	p.flush()

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Converted Cheats for %s\n\n", capitalize(filenameBase(filename)))
	for _, e := range p.entries {
		writeCheatEntry(&sb, e, tags)
	}
	return sb.String(), nil
}

func writeCheatEntry(sb *strings.Builder, e CheatEntry, tags []string) {
	command, placeholders := processCheatCommand(e.Command)

	fmt.Fprintf(sb, "## %s\n", e.Desc)
	writeHashtags(sb, tags)
	sb.WriteByte('\n')
	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)
	if block := emitTldrVarBlock(placeholders); block != "" {
		sb.WriteString(block)
	}
	sb.WriteByte('\n')
}

func processCheatCommand(command string) (string, []tldrPlaceholder) {
	naviRaws := extractPlaceholders(command, naviPlaceholderRe)
	tldrRaws := extractTldrPlaceholders(command)
	bracedRaws := extractCheatBracedPlaceholders(command)

	placeholders := make([]tldrPlaceholder, 0, len(naviRaws)+len(tldrRaws)+len(bracedRaws))
	for _, raw := range naviRaws {
		placeholders = append(placeholders, tldrPlaceholder{
			Raw:  raw,
			Name: sanitizeVarName(raw),
		})
	}
	placeholders = append(placeholders, classifyTldrPlaceholders(tldrRaws)...)
	for _, raw := range bracedRaws {
		placeholders = append(placeholders, tldrPlaceholder{
			Raw:  raw,
			Name: sanitizeVarName(raw),
		})
	}
	placeholders = uniqueCheatPlaceholders(placeholders)

	for _, ph := range placeholders {
		command = strings.ReplaceAll(command, "<"+ph.Raw+">", "$"+ph.Name)
		command = substituteTldrPlaceholder(command, ph.Raw, ph.Name)
		command = strings.ReplaceAll(command, "${"+ph.Raw+"}", "$"+ph.Name)
	}
	return command, placeholders
}

func extractCheatBracedPlaceholders(command string) []string {
	var list []string
	seen := make(map[string]struct{})
	for i := 0; i < len(command)-2; i++ {
		if command[i] == '\\' {
			i++
			continue
		}
		if command[i] != '$' || command[i+1] != '{' {
			continue
		}
		end := strings.IndexByte(command[i+2:], '}')
		if end == -1 {
			continue
		}
		raw := command[i+2 : i+2+end]
		if !isCheatBracedPlaceholderName(raw) {
			continue
		}
		if _, ok := seen[raw]; !ok {
			seen[raw] = struct{}{}
			list = append(list, raw)
		}
		i += end + 2
	}
	return list
}

func isCheatBracedPlaceholderName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func uniqueCheatPlaceholders(placeholders []tldrPlaceholder) []tldrPlaceholder {
	out := make([]tldrPlaceholder, 0, len(placeholders))
	byRaw := make(map[string]string)
	nameCount := make(map[string]int)
	for _, ph := range placeholders {
		if name, ok := byRaw[ph.Raw]; ok {
			ph.Name = name
			continue
		}
		uniq := nameCount[ph.Name]
		nameCount[ph.Name] = uniq + 1
		if uniq > 0 {
			ph.Name = fmt.Sprintf("%s_%d", ph.Name, uniq+1)
		}
		byRaw[ph.Raw] = ph.Name
		out = append(out, ph)
	}
	return out
}
