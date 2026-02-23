package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MapRemoteManagerState holds map remote manager modal state.
type MapRemoteManagerState struct {
	cursor int
}

func (m *Model) openMapRemoteManager() {
	m.activeModal = ModalMapRemoteManager
	m.mapRemoteMgr.cursor = 0
}

func (m *Model) closeMapRemoteManager() {
	m.activeModal = ModalNone
}

func (m *Model) updateMapRemoteManager(msg tea.KeyMsg) tea.Cmd {
	rules := m.vm.GetMapRemoteRules()

	switch msg.String() {
	case "q", "esc":
		m.closeMapRemoteManager()
	case "j", "down":
		if m.mapRemoteMgr.cursor < len(rules)-1 {
			m.mapRemoteMgr.cursor++
		}
	case "k", "up":
		if m.mapRemoteMgr.cursor > 0 {
			m.mapRemoteMgr.cursor--
		}
	case "enter", " ", "tab":
		if m.mapRemoteMgr.cursor < len(rules) {
			m.vm.ToggleMapRemoteRule(rules[m.mapRemoteMgr.cursor].ID)
		}
	case "d", "x", "backspace", "delete":
		if m.mapRemoteMgr.cursor < len(rules) {
			m.vm.RemoveMapRemoteRule(rules[m.mapRemoteMgr.cursor].ID)
			newRules := m.vm.GetMapRemoteRules()
			if m.mapRemoteMgr.cursor >= len(newRules) && m.mapRemoteMgr.cursor > 0 {
				m.mapRemoteMgr.cursor--
			}
		}
	case "e":
		if m.mapRemoteMgr.cursor < len(rules) {
			m.editMapRemoteRule(rules[m.mapRemoteMgr.cursor].ID)
		}
	case "a":
		m.openMapRemoteForm("", "")
	}
	return nil
}

func (m *Model) renderMapRemoteManager() string {
	rules := m.vm.GetMapRemoteRules()

	if len(rules) == 0 {
		var items []ModalListItem
		items = append(items, ModalListItem{Label: "No mappings configured", Selectable: false})
		items = append(items, ModalListItem{Label: "", Selectable: false})
		items = append(items, ModalListItem{Label: "Press 'a' to add a mapping", Selectable: false})
		return renderModalWithList(
			"Map Remote",
			items, -1, "a:add | Esc:close", 80,
		)
	}

	var items []ModalListItem
	enabledCount := 0
	for _, rule := range rules {
		icon := disabledItemStyle.Render("○")
		if rule.Enabled {
			icon = enabledItemStyle.Render("●")
			enabledCount++
		}
		items = append(items, ModalListItem{
			Label:      fmt.Sprintf("%s %s", icon, rule.Pattern),
			Secondary:  fmt.Sprintf("-> %s", rule.Replacement),
			Selectable: true,
		})
	}

	countStyle := lipgloss.NewStyle().Foreground(accentColor())
	footer := fmt.Sprintf("%s enabled | a:add | e:edit | Space:toggle | d:delete | Esc:close",
		countStyle.Render(fmt.Sprintf("%d/%d", enabledCount, len(rules))))
	return renderModalWithList(
		fmt.Sprintf("Map Remote (%d)", len(rules)),
		items, m.mapRemoteMgr.cursor, footer, 80,
	)
}

// openMapRemoteForm opens the form for adding/editing a map remote rule.
func (m *Model) openMapRemoteForm(pattern, remoteURL string) {
	m.activeModal = ModalMapRemoteForm
	m.mapRemoteForm = NewModalFormModel("Add Map Remote Rule", 70)
	m.mapRemoteForm.AddField("URL Pattern:", pattern)
	m.mapRemoteForm.AddField("Remote URL:", remoteURL)
	m.mapRemoteForm.SetButtons([]string{"Add", "Cancel"})
	m.mapRemoteEditID = 0 // add mode
}

func (m *Model) editMapRemoteRule(ruleID int) {
	rule := m.vm.GetMapRemoteRuleByID(ruleID)
	if rule == nil {
		return
	}
	m.activeModal = ModalMapRemoteForm
	m.mapRemoteForm = NewModalFormModel("Edit Map Remote Rule", 70)
	m.mapRemoteForm.AddField("URL Pattern:", rule.Pattern)
	m.mapRemoteForm.AddField("Remote URL:", rule.Replacement)
	m.mapRemoteForm.SetButtons([]string{"Save", "Cancel"})
	m.mapRemoteEditID = ruleID
}

func (m *Model) updateMapRemoteForm(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.openMapRemoteManager()
		return nil
	case "enter":
		if m.mapRemoteForm.FocusedButton() == 0 {
			pattern := strings.TrimSpace(m.mapRemoteForm.GetValue(0))
			remoteURL := strings.TrimSpace(m.mapRemoteForm.GetValue(1))
			if pattern != "" && remoteURL != "" {
				if m.mapRemoteEditID > 0 {
					m.vm.UpdateMapRemoteRule(m.mapRemoteEditID, pattern, remoteURL, "")
					m.statusMsg = fmt.Sprintf("Updated mapping: %s -> %s", pattern, remoteURL)
				} else {
					m.vm.AddMapRemoteRule(pattern, remoteURL, "")
					m.statusMsg = fmt.Sprintf("Added mapping: %s -> %s", pattern, remoteURL)
				}
			}
			m.openMapRemoteManager()
			return nil
		} else if m.mapRemoteForm.FocusedButton() == 1 {
			m.openMapRemoteManager()
			return nil
		}
	}
	return m.mapRemoteForm.Update(msg)
}

func (m *Model) renderMapRemoteForm() string {
	return m.mapRemoteForm.View()
}
