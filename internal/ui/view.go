package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"proxy-tui/internal/model"
)

// renderFilterBar renders the filter bar at the top.
func (m *Model) renderFilterBar() string {
	items := []struct {
		label string
		ft    model.FilterType
	}{
		{"1 All", model.FilterAll},
		{"2 Whitelist", model.FilterWhitelist},
		{"3 Starred", model.FilterStarred},
	}

	var parts []string
	for _, item := range items {
		if m.filterType == item.ft {
			parts = append(parts, filterActiveStyle().Render(item.label))
		} else {
			parts = append(parts, filterInactiveStyle.Render(item.label))
		}
	}

	// Custom filter
	customLabel := "/ Custom"
	if m.customPattern != "" {
		customLabel = "/ " + m.customPattern
	}
	if m.filterType == model.FilterCustom {
		parts = append(parts, filterActiveStyle().Render(customLabel))
	} else {
		parts = append(parts, filterInactiveStyle.Render(customLabel))
	}

	sep := filterSepStyle.Render("  ")
	bar := strings.Join(parts, sep)
	return filterBarStyle.Width(m.width).Render(bar)
}

// renderStatusBar renders the status bar at the bottom.
func (m *Model) renderStatusBar() string {
	left := m.statusMsg
	if left == "" {
		if m.focusedPanel == 0 {
			left = styleKeyHints([]keyHint{
				{"l", "local"},
				{"L", "local mgr"},
				{"r", "remote"},
				{"R", "remote mgr"},
				{"w", "whitelist"},
				{"W", "whitelist mgr"},
				{"c", "clear"},
				{"H", "expand"},
				{"?", "help"},
			})
		} else {
			left = styleKeyHints([]keyHint{
				{"j/k", "scroll"},
				{"T", "raw"},
				{"l", "local"},
				{"L", "local mgr"},
				{"r", "remote"},
				{"R", "remote mgr"},
				{"w", "whitelist"},
				{"W", "whitelist mgr"},
				{"H", "expand"},
				{"?", "help"},
			})
		}
	}

	right := m.address

	leftWidth := m.width - lipgloss.Width(right) - 2
	if leftWidth < 0 {
		leftWidth = 0
	}

	leftStr := statusBarStyle.Width(leftWidth).Render(" " + left)
	rightStr := addressBarStyle.Render(right + " ")

	return lipgloss.JoinHorizontal(lipgloss.Top, leftStr, rightStr)
}

type keyHint struct {
	key  string
	desc string
}

func styleKeyHints(hints []keyHint) string {
	var parts []string
	keyStyle := lipgloss.NewStyle().Foreground(accentColor())
	descStyle := lipgloss.NewStyle().Foreground(colorSubtle)
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.key)+descStyle.Render(":"+h.desc))
	}
	return strings.Join(parts, " ")
}

// View renders the full TUI.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Filter bar (1 line + 1 border line)
	filterBar := m.renderFilterBar()
	filterBarHeight := lipgloss.Height(filterBar)

	// Status bar (1 line)
	statusBar := m.renderStatusBar()

	// Main content area height
	contentHeight := m.height - filterBarHeight - 1

	// Compute panel dimensions
	borderH := 2 // top + bottom border
	innerH := contentHeight - borderH
	if innerH < 1 {
		innerH = 1
	}

	var mainContent string

	if m.expanded {
		// Expanded: single panel full width
		panelWidth := m.width
		innerW := panelWidth - 2
		if innerW < 1 {
			innerW = 1
		}

		if m.focusedPanel == 0 {
			m.requestList.SetSize(innerW, innerH)
			body := m.requestList.View()
			title := m.requestList.Title()
			mainContent = renderPanel(body, title, panelWidth, contentHeight, true)
		} else {
			m.detailView.SetSize(innerW, innerH)
			body := m.detailView.View()
			title := m.detailView.Title()
			mainContent = renderPanel(body, title, panelWidth, contentHeight, true)
		}
	} else {
		// Normal: two panels side by side
		leftWidth := m.width * 65 / 100
		rightWidth := m.width - leftWidth

		leftInnerW := leftWidth - 2
		rightInnerW := rightWidth - 2
		if leftInnerW < 1 {
			leftInnerW = 1
		}
		if rightInnerW < 1 {
			rightInnerW = 1
		}

		m.requestList.SetSize(leftInnerW, innerH)
		m.detailView.SetSize(rightInnerW, innerH)

		leftBody := m.requestList.View()
		leftTitle := m.requestList.Title()
		rightBody := m.detailView.View()
		rightTitle := m.detailView.Title()

		leftPanel := renderPanel(leftBody, leftTitle, leftWidth, contentHeight, m.focusedPanel == 0)
		rightPanel := renderPanel(rightBody, rightTitle, rightWidth, contentHeight, m.focusedPanel == 1)

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	}

	// Stack everything vertically
	full := lipgloss.JoinVertical(lipgloss.Left, filterBar, mainContent, statusBar)

	// Overlay modal if active
	if m.activeModal != ModalNone {
		modalContent := m.renderActiveModal()
		full = placeOverlay(m.width, m.height, full, modalContent)
	}

	// Overlay help if active
	if m.showHelp {
		helpContent := m.renderHelp()
		full = placeOverlay(m.width, m.height, full, helpContent)
	}

	return full
}

// renderPanel renders a bordered panel with title by manually constructing the border.
func renderPanel(body, title string, width, height int, focused bool) string {
	borderFg := colorDim
	titleRendered := panelTitleDimStyle.Render(title)
	if focused {
		borderFg = accentColor()
		titleRendered = panelTitleStyle().Render(title)
	}

	bdr := lipgloss.RoundedBorder()
	bs := lipgloss.NewStyle().Foreground(borderFg)

	innerW := width - 2
	innerH := height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// Top border: ╭─ Title ──────╮
	titleDisplayW := lipgloss.Width(titleRendered)
	dashesAfter := innerW - titleDisplayW - 3 // 1 dash before + 2 spaces around title
	if dashesAfter < 0 {
		dashesAfter = 0
	}
	topLine := bs.Render(bdr.TopLeft+bdr.Top) +
		" " + titleRendered + " " +
		bs.Render(strings.Repeat(bdr.Top, dashesAfter)+bdr.TopRight)

	// Body lines with side borders
	bodyLines := strings.Split(body, "\n")
	for len(bodyLines) < innerH {
		bodyLines = append(bodyLines, strings.Repeat(" ", innerW))
	}
	if len(bodyLines) > innerH {
		bodyLines = bodyLines[:innerH]
	}

	var rows []string
	rows = append(rows, topLine)
	for _, line := range bodyLines {
		lineW := lipgloss.Width(line)
		pad := ""
		if lineW < innerW {
			pad = strings.Repeat(" ", innerW-lineW)
		}
		rows = append(rows, bs.Render(bdr.Left)+line+pad+bs.Render(bdr.Right))
	}

	// Bottom border
	bottomLine := bs.Render(bdr.BottomLeft + strings.Repeat(bdr.Bottom, innerW) + bdr.BottomRight)
	rows = append(rows, bottomLine)

	return strings.Join(rows, "\n")
}

// placeOverlay places centered modal content over background.
func placeOverlay(width, height int, background, modal string) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}),
	)
}
