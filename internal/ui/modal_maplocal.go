package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MapLocalManagerState holds map local manager modal state.
type MapLocalManagerState struct {
	cursor int
}

func (m *Model) openMapLocalManager() {
	m.activeModal = ModalMapLocalManager
	m.mapLocalMgr.cursor = 0
}

func (m *Model) closeMapLocalManager() {
	m.activeModal = ModalNone
}

func (m *Model) updateMapLocalManager(msg tea.KeyMsg) tea.Cmd {
	rules := m.vm.GetMapLocalRules()

	switch msg.String() {
	case "q", "esc":
		m.closeMapLocalManager()
	case "j", "down":
		if m.mapLocalMgr.cursor < len(rules)-1 {
			m.mapLocalMgr.cursor++
		}
	case "k", "up":
		if m.mapLocalMgr.cursor > 0 {
			m.mapLocalMgr.cursor--
		}
	case "enter", " ", "tab":
		if m.mapLocalMgr.cursor < len(rules) {
			m.vm.ToggleMapLocalRule(rules[m.mapLocalMgr.cursor].ID)
		}
	case "d", "x", "backspace", "delete":
		if m.mapLocalMgr.cursor < len(rules) {
			m.vm.RemoveMapLocalRule(rules[m.mapLocalMgr.cursor].ID)
			newRules := m.vm.GetMapLocalRules()
			if m.mapLocalMgr.cursor >= len(newRules) && m.mapLocalMgr.cursor > 0 {
				m.mapLocalMgr.cursor--
			}
		}
	case "e":
		if m.mapLocalMgr.cursor < len(rules) {
			return m.editMapLocalFile(rules[m.mapLocalMgr.cursor].Replacement)
		}
	case "a":
		m.openMapLocalForm()
	}
	return nil
}

// editMapLocalFile opens the file in $EDITOR using tea.ExecProcess.
func (m *Model) editMapLocalFile(localPath string) tea.Cmd {
	if strings.HasPrefix(localPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			localPath = home + localPath[1:]
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, localPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func (m *Model) renderMapLocalManager() string {
	rules := m.vm.GetMapLocalRules()

	if len(rules) == 0 {
		var items []ModalListItem
		items = append(items, ModalListItem{Label: "No mappings configured", Selectable: false})
		items = append(items, ModalListItem{Label: "", Selectable: false})
		items = append(items, ModalListItem{Label: "Press 'a' to add a mapping", Selectable: false})
		return renderModalWithList(
			"Map Local",
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
	footer := fmt.Sprintf("%s enabled | a:add | e:edit file | Space:toggle | d:delete | Esc:close",
		countStyle.Render(fmt.Sprintf("%d/%d", enabledCount, len(rules))))
	return renderModalWithList(
		fmt.Sprintf("Map Local (%d)", len(rules)),
		items, m.mapLocalMgr.cursor, footer, 80,
	)
}

// openMapLocalForm opens the form for adding a new map local rule.
func (m *Model) openMapLocalForm() {
	m.activeModal = ModalMapLocalForm
	m.mapLocalForm = NewModalFormModel("Add Map Local Rule", 70)
	m.mapLocalForm.AddField("URL Pattern:", "")
	m.mapLocalForm.AddField("Local Path:", "")
	m.mapLocalForm.SetButtons([]string{"Add", "Cancel"})
}

// openMapLocalPatternInput opens a pattern input to quick-map the selected flow.
func (m *Model) openMapLocalPatternInput() {
	flow := m.vm.GetSelectedFlow()
	if flow == nil {
		m.statusMsg = "No request selected"
		return
	}
	if flow.Response == nil {
		m.statusMsg = "No response to map"
		return
	}
	if flow.Tunneled {
		m.statusMsg = "Cannot map tunneled connections"
		return
	}

	m.activeModal = ModalMapLocalPattern
	m.mapLocalPatternInput = textinput.New()
	m.mapLocalPatternInput.SetValue(flow.Request.URL)
	m.mapLocalPatternInput.Focus()
	m.mapLocalPatternInput.Width = 70
}

func (m *Model) updateMapLocalForm(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.openMapLocalManager()
		return nil
	case "enter":
		if m.mapLocalForm.FocusedButton() == 0 {
			pattern := strings.TrimSpace(m.mapLocalForm.GetValue(0))
			localPath := strings.TrimSpace(m.mapLocalForm.GetValue(1))
			if pattern != "" && localPath != "" {
				m.vm.AddMapLocalRule(pattern, localPath, 200, "", "")
				m.statusMsg = fmt.Sprintf("Added mapping: %s -> %s", pattern, localPath)
			}
			m.openMapLocalManager()
			return nil
		} else if m.mapLocalForm.FocusedButton() == 1 {
			m.openMapLocalManager()
			return nil
		}
	}
	return m.mapLocalForm.Update(msg)
}

func (m *Model) updateMapLocalPattern(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.activeModal = ModalNone
		return nil
	case "enter":
		pattern := strings.TrimSpace(m.mapLocalPatternInput.Value())
		if pattern != "" {
			m.createMapLocalWithPattern(pattern)
		}
		m.activeModal = ModalNone
		return nil
	}

	var cmd tea.Cmd
	m.mapLocalPatternInput, cmd = m.mapLocalPatternInput.Update(msg)
	return cmd
}

func (m *Model) renderMapLocalForm() string {
	return m.mapLocalForm.View()
}

func (m *Model) renderMapLocalPattern() string {
	return renderInputModal("URL Pattern", m.mapLocalPatternInput, 80)
}
