package convert

import (
	"fmt"
	"strings"
)

type tldrExample struct {
	Desc    string
	Command string
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

	if strings.HasPrefix(line, "- ") {
		p.currentDesc = strings.TrimSuffix(strings.TrimSpace(line[2:]), ":")
		return index
	}

	if p.currentDesc == "" {
		return index
	}

	var command string
	if strings.HasPrefix(line, "`") && strings.HasSuffix(line, "`") && len(line) >= 2 {
		command = line[1 : len(line)-1]
	} else if strings.HasPrefix(line, "```") {
		command, index = consumeCodeBlock(lines, index+1)
	}

	if command != "" {
		p.examples = append(p.examples, tldrExample{Desc: p.currentDesc, Command: command})
		p.currentDesc = ""
	}
	return index
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
	command, placeholders := rewriteTldr(ex.Command)
	fmt.Fprintf(sb, "## %s\n\n", ex.Desc)
	fmt.Fprintf(sb, "```sh\n%s\n```\n", command)
	sb.WriteString(formatHeaderVarsBlock(placeholders))
	sb.WriteByte('\n')
}
