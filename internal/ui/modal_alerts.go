package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// AlertManagerState holds alert manager modal state.
type AlertManagerState struct {
	cursor int
}

func (m *Model) openAlertManager() {
	m.activeModal = ModalAlertManager
	m.alertMgr.cursor = 0
}

func (m *Model) closeAlertManager() {
	m.activeModal = ModalNone
}

func (m *Model) updateAlertManager(msg tea.KeyMsg) tea.Cmd {
	rules := m.vm.GetAlertRules()

	switch msg.String() {
	case "q", "esc":
		m.closeAlertManager()
	case "j", "down":
		if m.alertMgr.cursor < len(rules)-1 {
			m.alertMgr.cursor++
		}
	case "k", "up":
		if m.alertMgr.cursor > 0 {
			m.alertMgr.cursor--
		}
	case "enter", " ", "tab":
		if m.alertMgr.cursor < len(rules) {
			m.vm.ToggleAlertRule(m.alertMgr.cursor)
		}
	}
	return nil
}

func (m *Model) renderAlertManager() string {
	rules := m.vm.GetAlertRules()

	if len(rules) == 0 {
		var items []ModalListItem
		items = append(items, ModalListItem{Label: "No alert rules configured", Selectable: false})
		return renderModalWithList("Alert Settings", items, -1, "Esc:close", 50)
	}

	var items []ModalListItem
	for _, rule := range rules {
		label := formatAlertRule(rule)
		status := fmtEnabled(rule.Enabled)
		items = append(items, ModalListItem{
			Label:      fmt.Sprintf("%-20s %s", label, status),
			Secondary:  "Enter/Space to toggle",
			Selectable: true,
		})
	}

	return renderModalWithList("Alert Settings", items, m.alertMgr.cursor, "Esc:close", 50)
}
