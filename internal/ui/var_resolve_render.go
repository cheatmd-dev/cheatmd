package ui

import (
	"fmt"
	"strings"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
	"github.com/cheatmd-dev/cheatmd/pkg/executor"
)

// renderVarResolve renders the variable resolution view.
func (m *mainModel) renderVarResolve() string {
	if m.varState == nil {
		return ""
	}

	width := max(m.width, 80)
	height := m.height
	if height < 1 {
		height = 24
	}

	b := getBuilder()
	defer putBuilder(b)

	header := m.renderVarHeader(width)
	headerLines := countLines(header)

	availableForBottom := max(height-headerLines, 5)
	bottom := m.renderVarBottomWithHeight(width, availableForBottom)
	bottomLines := countLines(bottom)

	padding := max(height-headerLines-bottomLines, 0)

	b.WriteString(header)
	b.WriteString(strings.Repeat("\n", padding))
	b.WriteString(bottom)

	return b.String()
}

// renderVarBottomWithHeight renders the options list and input with a max height.
func (m *mainModel) renderVarBottomWithHeight(width int, maxHeight int) string {
	b := getBuilder()
	defer putBuilder(b)

	b.WriteString(styles.Divider.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Fixed lines: top divider(1) + bottom divider(1) + info line(1) + input(1) = 4
	fixedLines := 4

	availableForList := max(maxHeight-fixedLines, 1)

	if m.varState.showPathCompletions && len(m.varState.pathCompletions) > 0 {
		listHeight := min(availableForList, min(10, len(m.varState.pathCompletions)))
		for i := 0; i < listHeight; i++ {
			candidate := m.varState.pathCompletions[i]
			b.WriteString("  ")
			b.WriteString(styles.Command.Render(candidate.Display))
			b.WriteString("\n")
		}
	} else if !m.varState.isPromptOnly && m.varState.picker != nil && len(m.varState.picker.Filtered) > 0 {
		listHeight := min(availableForList, min(10, len(m.varState.picker.Filtered)))
		start, end := scrollWindow(m.varState.picker.Cursor, len(m.varState.picker.Filtered), listHeight, &m.varState.picker.Offset)

		for i := start; i < end; i++ {
			opt := m.varState.picker.Filtered[i]

			checkbox := ""
			if m.varState.selectOpts.Multi {
				vs := &m.varState.vars[m.varState.currentIdx]
				if vs.multiSelectedSet[opt.Original] {
					checkbox = styles.Command.Render("[x] ")
				} else {
					checkbox = styles.Dim.Render("[ ] ")
				}
			}

			if i == m.varState.picker.Cursor {
				b.WriteString(styles.Cursor.Render("▶ "))
				b.WriteString(checkbox)
				b.WriteString(styles.Selected.Render(styles.Command.Render(opt.Display)))
			} else {
				b.WriteString("  ")
				b.WriteString(checkbox)
				b.WriteString(styles.Command.Render(opt.Display))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(styles.Divider.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	if m.varState.showPathCompletions && len(m.varState.pathCompletions) > 0 {
		b.WriteString(styles.Dim.Render(fmt.Sprintf("  %d path matches", len(m.varState.pathCompletions))))
		b.WriteString(" • ")
	} else if !m.varState.isPromptOnly && m.varState.picker != nil && len(m.varState.picker.Filtered) > 0 {
		b.WriteString(styles.Dim.Render(fmt.Sprintf("  %d options", len(m.varState.picker.Filtered))))
		b.WriteString(" • ")
	}
	b.WriteString(styles.Dim.Render("ESC back"))
	if m.varState.selectOpts.Multi {
		b.WriteString(" • ")
		b.WriteString(styles.Dim.Render("Space toggle"))
	}
	b.WriteString(" • ")
	b.WriteString(styles.Dim.Render("Enter accept"))
	if !m.varState.isPromptOnly {
		b.WriteString(" • ")
		b.WriteString(styles.Dim.Render("Alt+Enter bypass"))
	}
	b.WriteString("\n")
	b.WriteString(m.textInput.View())

	return b.String()
}

// renderVarHeader renders the progress header for variable resolution.
func (m *mainModel) renderVarHeader(width int) string {
	if m.varState == nil {
		return ""
	}

	b := getBuilder()
	defer putBuilder(b)

	progressCmd := m.varState.cheat.Command
	for i, vs := range m.varState.vars {
		if vs.resolved {
			progressCmd = executor.ReplaceVar(progressCmd, vs.def.Name, styles.Header.Render(vs.value), config.Get().VarSyntax)
		} else if i == m.varState.currentIdx {
			displayStr := formatVarName(m.varState.cheat.Command, vs.def.Name)
			progressCmd = executor.ReplaceVar(progressCmd, vs.def.Name, styles.Cursor.Render(displayStr), config.Get().VarSyntax)
		}
	}
	b.WriteString(progressCmd)
	b.WriteString("\n")

	for i, vs := range m.varState.vars {
		displayStr := formatVarName(m.varState.cheat.Command, vs.def.Name)
		if vs.resolved {
			b.WriteString(styles.Command.Render("✓"))
			b.WriteString(" ")
			b.WriteString(styles.Dim.Render(displayStr))
			b.WriteString(" = ")
			b.WriteString(styles.Header.Render(vs.value))
		} else if i == m.varState.currentIdx {
			b.WriteString(styles.Cursor.Render("▶ " + displayStr))
		} else {
			b.WriteString(styles.Dim.Render("○ " + displayStr))
		}
		b.WriteString("\n")
	}

	if m.varState.customHeader != "" {
		b.WriteString("\n")
		b.WriteString(styles.Header.Render(m.varState.customHeader))
		b.WriteString("\n")
	}

	b.WriteString(styles.Divider.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	return b.String()
}

// formatVarName returns the variable name formatted according to how it appears in the command,
// or defaults based on the syntax config.
func formatVarName(cmd string, name string) string {
	if config.Get().VarSyntax == "angle" {
		return "<" + name + ">"
	} else if config.Get().VarSyntax == "both" {
		if strings.Contains(cmd, "<"+name+">") {
			return "<" + name + ">"
		}
	}
	return "$" + name
}
