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
	command, placeholders := rewriteBoth(e.Command)

	fmt.Fprintf(sb, "## %s\n", e.Desc)
	writeHashtags(sb, tags)
	sb.WriteByte('\n')
	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)
	sb.WriteString(formatHeaderVarsBlock(placeholders))
	sb.WriteByte('\n')
}
