package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WhitelistManagerState holds the whitelist manager modal state.
type WhitelistManagerState struct {
	cursor int
}

func (m *Model) openWhitelistManager() {
	m.activeModal = ModalWhitelistManager
	m.whitelistMgr.cursor = 0
}

func (m *Model) closeWhitelistManager() {
	m.activeModal = ModalNone
}

func (m *Model) updateWhitelistManager(msg tea.KeyMsg) tea.Cmd {
	patterns := m.vm.GetWhitelistPatterns()

	switch msg.String() {
	case "q", "esc":
		m.closeWhitelistManager()
	case "j", "down":
		if m.whitelistMgr.cursor < len(patterns)-1 {
			m.whitelistMgr.cursor++
		}
	case "k", "up":
		if m.whitelistMgr.cursor > 0 {
			m.whitelistMgr.cursor--
		}
	case "enter", " ", "tab":
		if m.whitelistMgr.cursor < len(patterns) {
			m.vm.ToggleWhitelistPattern(patterns[m.whitelistMgr.cursor].Pattern)
		}
	case "d", "x", "backspace", "delete":
		if m.whitelistMgr.cursor < len(patterns) {
			m.vm.RemoveWhitelistPattern(patterns[m.whitelistMgr.cursor].Pattern)
			if m.whitelistMgr.cursor >= len(m.vm.GetWhitelistPatterns()) && m.whitelistMgr.cursor > 0 {
				m.whitelistMgr.cursor--
			}
		}
	case "e":
		if m.whitelistMgr.cursor < len(patterns) {
			m.openWhitelistEdit(patterns[m.whitelistMgr.cursor].Pattern)
		}
	case "a":
		m.openWhitelistInput("")
	}
	return nil
}

func (m *Model) renderWhitelistManager() string {
	patterns := m.vm.GetWhitelistPatterns()

	if len(patterns) == 0 {
		var items []ModalListItem
		items = append(items, ModalListItem{Label: "No patterns configured", Selectable: false})
		items = append(items, ModalListItem{Label: "", Selectable: false})
		items = append(items, ModalListItem{Label: "Press 'a' to add a pattern", Selectable: false})
		return renderModalWithList(
			"Whitelist Manager",
			items, -1, "a:add | Esc:close", 70,
		)
	}

	var items []ModalListItem
	enabledCount := 0
	for _, hp := range patterns {
		icon := disabledItemStyle.Render("○")
		if hp.Enabled {
			icon = enabledItemStyle.Render("●")
			enabledCount++
		}
		items = append(items, ModalListItem{
			Label:      fmt.Sprintf("%s %s", icon, hp.Pattern),
			Selectable: true,
		})
	}

	countStyle := lipgloss.NewStyle().Foreground(accentColor())
	footer := fmt.Sprintf("%s enabled | a:add | e:edit | Space:toggle | d:delete | Esc:close",
		countStyle.Render(fmt.Sprintf("%d/%d", enabledCount, len(patterns))))
	return renderModalWithList(
		fmt.Sprintf("Whitelist Manager (%d)", len(patterns)),
		items, m.whitelistMgr.cursor, footer, 70,
	)
}

// openWhitelistInput opens the whitelist input modal.
func (m *Model) openWhitelistInput(prefill string) {
	m.activeModal = ModalWhitelistInput
	m.whitelistInput = textinput.New()
	m.whitelistInput.Placeholder = "*.example.com"
	m.whitelistInput.SetValue(prefill)
	m.whitelistInput.Focus()
	m.whitelistInput.Width = 60
	m.whitelistEditOld = "" // add mode
}

// openWhitelistEdit opens the whitelist input in edit mode.
func (m *Model) openWhitelistEdit(oldPattern string) {
	m.activeModal = ModalWhitelistInput
	m.whitelistInput = textinput.New()
	m.whitelistInput.Placeholder = "*.example.com"
	m.whitelistInput.SetValue(oldPattern)
	m.whitelistInput.Focus()
	m.whitelistInput.Width = 60
	m.whitelistEditOld = oldPattern // edit mode
}

func (m *Model) updateWhitelistInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if m.whitelistEditOld != "" {
			m.openWhitelistManager()
		} else {
			m.activeModal = ModalNone
		}
		return nil
	case "enter":
		pattern := m.whitelistInput.Value()
		if pattern != "" {
			if m.whitelistEditOld != "" {
				m.vm.EditWhitelistPattern(m.whitelistEditOld, pattern)
				m.statusMsg = fmt.Sprintf("Updated pattern: %s -> %s", m.whitelistEditOld, pattern)
			} else {
				m.vm.AddWhitelistPattern(pattern)
				m.statusMsg = fmt.Sprintf("Added pattern: %s", pattern)
			}
		}
		m.openWhitelistManager()
		return nil
	}

	var cmd tea.Cmd
	m.whitelistInput, cmd = m.whitelistInput.Update(msg)
	return cmd
}

func (m *Model) renderWhitelistInput() string {
	title := "Add Pattern"
	if m.whitelistEditOld != "" {
		title = "Edit Pattern"
	}
	return renderInputModal(title, m.whitelistInput, 70)
}
