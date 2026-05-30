package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cheatmd-dev/cheatmd/pkg/registry"
)

// packPickerModel is a standalone multi-select checklist used by first-run
// setup to let the user choose which cheat packs to install. It reuses the
// generic Picker for cursor/scroll bookkeeping and the global styles for
// coloring.
type packPickerModel struct {
	picker    *Picker[registry.Pack]
	selected  map[int]bool // index into picker.Items -> checked
	installed map[string]bool
	cancelled bool
}

// RunPackPicker shows an interactive checklist of packs and returns those the
// user chose to install. Packs flagged Starter are pre-checked. Returns an
// empty slice if the user cancels (q/esc/ctrl+c).
func RunPackPicker(packs []registry.Pack, installed map[string]bool) ([]registry.Pack, error) {
	if len(packs) == 0 {
		return nil, nil
	}

	sortedPacks := make([]registry.Pack, len(packs))
	copy(sortedPacks, packs)
	sort.SliceStable(sortedPacks, func(i, j int) bool {
		if sortedPacks[i].Official != sortedPacks[j].Official {
			return sortedPacks[i].Official // true comes before false
		}
		return sortedPacks[i].Name < sortedPacks[j].Name
	})

	selected := make(map[int]bool, len(sortedPacks))

	m := &packPickerModel{
		// No filtering needed; the filter func is a no-op match-all.
		picker:    NewPicker(sortedPacks, func(registry.Pack, []string) bool { return true }),
		selected:  selected,
		installed: installed,
	}

	ttyIn, ttyOut, cleanup := getTTY()
	RefreshStyles()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(ttyOut), tea.WithInput(ttyIn))
	finalModel, err := p.Run()
	cleanup()
	if err != nil {
		return nil, err
	}

	res := finalModel.(*packPickerModel)
	if res.cancelled {
		return nil, nil
	}

	var chosen []registry.Pack
	for i, pack := range res.picker.Items {
		if res.selected[i] {
			chosen = append(chosen, pack)
		}
	}
	return chosen, nil
}

func (m *packPickerModel) Init() tea.Cmd { return nil }

func (m *packPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.picker.HandleKey(msg) {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		case " ":
			// Toggle the checkbox for the highlighted row.
			idx := m.itemIndex(m.picker.Cursor)
			if idx >= 0 {
				m.selected[idx] = !m.selected[idx]
			}
			return m, nil
		case "a":
			// Select all.
			for i := range m.picker.Items {
				m.selected[i] = true
			}
			return m, nil
		case "n":
			// Select none.
			m.selected = make(map[int]bool, len(m.picker.Items))
			return m, nil
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

// itemIndex maps a cursor position in the (unfiltered) list back to the index
// into picker.Items. Since the filter is match-all, these are equal, but we
// resolve via Selected to stay correct if filtering is added later.
func (m *packPickerModel) itemIndex(cursor int) int {
	if cursor < 0 || cursor >= len(m.picker.Filtered) {
		return -1
	}
	target := m.picker.Filtered[cursor]
	for i, it := range m.picker.Items {
		if it.Name == target.Name && it.Repo == target.Repo {
			return i
		}
	}
	return -1
}

func (m *packPickerModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Header.Render("Install cheat packs"))
	b.WriteString("\n")
	b.WriteString(styles.Dim.Render("Choose cheat packs to install. Space toggles, Enter confirms."))
	b.WriteString("\n\n")

	for i, pack := range m.picker.Filtered {
		b.WriteString(m.renderRow(i, pack))
		b.WriteString("\n")
	}

	footer := fmt.Sprintf("%d selected · a: all · n: none · q/esc: cancel", m.selectedCount())
	b.WriteString("\n")
	b.WriteString(styles.Dim.Render(footer))
	b.WriteString("\n")

	return b.String()
}

// renderRow formats a single pack row: cursor marker, checkbox, name, and
// description, with the highlighted row's background applied.
func (m *packPickerModel) renderRow(i int, pack registry.Pack) string {
	idx := m.itemIndex(i)
	highlighted := i == m.picker.Cursor

	box := "[ ]"
	if idx >= 0 && m.selected[idx] {
		box = "[x]"
	}

	cursor := "  "
	if highlighted {
		cursor = styles.Cursor.Render("▶ ")
	}

	status := ""
	if m.installed[pack.Name] {
		status = styles.Dim.Render(" [installed]")
	}

	tag := "[community]"
	if pack.Official {
		tag = "[official]"
	}

	line := fmt.Sprintf("%s%s %s %s%s", cursor, box, styles.Dim.Render(tag), styles.Command.Render(pack.Name), status)
	if pack.Description != "" {
		line += "  " + styles.Desc.Render(pack.Description)
	}
	if highlighted {
		line = styles.Selected.Render(line)
	}
	return line
}

// selectedCount returns how many packs are currently checked.
func (m *packPickerModel) selectedCount() int {
	n := 0
	for _, checked := range m.selected {
		if checked {
			n++
		}
	}
	return n
}
