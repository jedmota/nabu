package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHelp renders a custom help overlay with a clean two-column layout.
func (m *Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(accentColor()).Bold(true).Width(8).Align(lipgloss.Right)
	descStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	sectionStyle := lipgloss.NewStyle().Foreground(colorWhite).Bold(true).MarginTop(1)

	type entry struct {
		key  string
		desc string
	}
	type section struct {
		title   string
		entries []entry
	}

	sections := []section{
		{"General", []entry{
			{"q", "Quit"},
			{"Tab", "Toggle focus"},
			{"H", "Expand panel"},
			{"?", "Toggle help"},
			{"p", "Pause/resume"},
			{"t", "Cycle theme"},
		}},
		{"Navigation", []entry{
			{"j/k", "Up/down"},
			{"gg", "Go to top"},
			{"G", "Go to bottom"},
			{"PgUp", "Page up"},
			{"PgDn", "Page down"},
		}},
		{"Filters", []entry{
			{"1", "All"},
			{"2", "Whitelist"},
			{"3", "Starred"},
			{"/", "Custom search"},
		}},
		{"Whitelist", []entry{
			{"w", "Add pattern"},
			{"W", "Manager"},
			{"C", "Clear all"},
		}},
		{"Mapping", []entry{
			{"l", "Map to local"},
			{"L", "Local manager"},
			{"r", "Map to remote"},
			{"R", "Remote manager"},
		}},
		{"Actions", []entry{
			{".", "Replay request"},
			{"x", "Copy as cURL"},
			{"s", "Star selected"},
			{"S", "Star all listed"},
			{"c", "Clear flows"},
			{"T", "Toggle raw/pretty"},
		}},
		{"Import/Export", []entry{
			{"e", "Export HAR"},
			{"E", "Export all HAR"},
			{"i", "Import HAR"},
			{"a", "Alert settings"},
		}},
	}

	// Split into two columns of sections
	mid := (len(sections) + 1) / 2
	leftSections := sections[:mid]
	rightSections := sections[mid:]

	renderColumn := func(secs []section) string {
		var sb strings.Builder
		for _, sec := range secs {
			sb.WriteString(sectionStyle.Render(sec.title))
			sb.WriteByte('\n')
			for _, e := range sec.entries {
				sb.WriteString(keyStyle.Render(e.key) + "  " + descStyle.Render(e.desc))
				sb.WriteByte('\n')
			}
		}
		return sb.String()
	}

	leftCol := lipgloss.NewStyle().Width(30).Render(renderColumn(leftSections))
	rightCol := lipgloss.NewStyle().Width(30).Render(renderColumn(rightSections))

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "   ", rightCol)

	content := modalTitleStyle().Render("Keybindings") + "  " + modalHintStyle.Render("? to close") + "\n" + columns

	return modalStyle().Width(68).Render(content)
}
