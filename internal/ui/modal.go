package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModalType represents which modal is currently active.
type ModalType int

const (
	ModalNone ModalType = iota
	ModalSearch
	ModalWhitelistInput
	ModalWhitelistManager
	ModalMapLocalPattern
	ModalMapLocalManager
	ModalMapLocalForm
	ModalMapRemoteManager
	ModalMapRemoteForm
	ModalAlertManager
	ModalImportHAR
)

// ModalListModel is a reusable list with cursor selection for modals.
type ModalListModel struct {
	items  []ModalListItem
	cursor int
	title  string
	footer string
	width  int
}

// ModalListItem represents one item in a modal list.
type ModalListItem struct {
	Label      string
	Secondary  string
	Enabled    bool
	Selectable bool
}

func NewModalListModel(title string, width int) ModalListModel {
	return ModalListModel{
		title: title,
		width: width,
	}
}

func (ml *ModalListModel) SetItems(items []ModalListItem) {
	ml.items = items
	if ml.cursor >= len(items) && len(items) > 0 {
		ml.cursor = len(items) - 1
	}
}

func (ml *ModalListModel) MoveUp() {
	for i := ml.cursor - 1; i >= 0; i-- {
		if ml.items[i].Selectable {
			ml.cursor = i
			return
		}
	}
}

func (ml *ModalListModel) MoveDown() {
	for i := ml.cursor + 1; i < len(ml.items); i++ {
		if ml.items[i].Selectable {
			ml.cursor = i
			return
		}
	}
}

func (ml *ModalListModel) SelectedIndex() int {
	if ml.cursor >= 0 && ml.cursor < len(ml.items) && ml.items[ml.cursor].Selectable {
		return ml.cursor
	}
	return -1
}

func (ml *ModalListModel) View() string {
	var sb strings.Builder

	if ml.title != "" {
		sb.WriteString(modalTitleStyle().Render(ml.title))
		sb.WriteString("\n\n")
	}

	for i, item := range ml.items {
		if !item.Selectable {
			sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(item.Label))
		} else if i == ml.cursor {
			sb.WriteString(listCursorStyle().Render("▌ ") + listSelectedStyle().Render(item.Label))
		} else {
			sb.WriteString("  " + item.Label)
		}
		if item.Secondary != "" {
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render("  " + item.Secondary))
		}
		sb.WriteString("\n")
	}

	if ml.footer != "" {
		sb.WriteString("\n")
		sb.WriteString(modalHintStyle.Render(ml.footer))
	}

	return modalStyle().Width(ml.width).Render(sb.String())
}

// FormField represents an input field in a modal form.
type FormField struct {
	Label string
	Input textinput.Model
}

// ModalFormModel is a reusable form with multiple text inputs.
type ModalFormModel struct {
	title    string
	fields   []FormField
	focused  int
	width    int
	buttons  []string
	btnFocus int // -1 means fields are focused, 0+ means buttons
}

func NewModalFormModel(title string, width int) ModalFormModel {
	return ModalFormModel{
		title:    title,
		width:    width,
		btnFocus: -1,
	}
}

func (mf *ModalFormModel) AddField(label, value string) {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.SetValue(value)
	ti.CharLimit = 500
	ti.Width = mf.width - 6
	if len(mf.fields) == 0 {
		ti.Focus()
	}
	mf.fields = append(mf.fields, FormField{Label: label, Input: ti})
}

func (mf *ModalFormModel) SetButtons(buttons []string) {
	mf.buttons = buttons
}

func (mf *ModalFormModel) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "down":
			return mf.focusNext()
		case "shift+tab", "up":
			return mf.focusPrev()
		}
	}

	if mf.btnFocus < 0 && len(mf.fields) > 0 {
		var cmd tea.Cmd
		mf.fields[mf.focused].Input, cmd = mf.fields[mf.focused].Input.Update(msg)
		return cmd
	}
	return nil
}

func (mf *ModalFormModel) focusNext() tea.Cmd {
	if mf.btnFocus < 0 {
		if mf.focused < len(mf.fields)-1 {
			mf.fields[mf.focused].Input.Blur()
			mf.focused++
			return mf.fields[mf.focused].Input.Focus()
		}
		if len(mf.buttons) > 0 {
			mf.fields[mf.focused].Input.Blur()
			mf.btnFocus = 0
		}
	} else if mf.btnFocus < len(mf.buttons)-1 {
		mf.btnFocus++
	}
	return nil
}

func (mf *ModalFormModel) focusPrev() tea.Cmd {
	if mf.btnFocus >= 0 {
		if mf.btnFocus > 0 {
			mf.btnFocus--
		} else {
			mf.btnFocus = -1
			if len(mf.fields) > 0 {
				mf.focused = len(mf.fields) - 1
				return mf.fields[mf.focused].Input.Focus()
			}
		}
	} else if mf.focused > 0 {
		mf.fields[mf.focused].Input.Blur()
		mf.focused--
		return mf.fields[mf.focused].Input.Focus()
	}
	return nil
}

func (mf *ModalFormModel) FocusedButton() int {
	return mf.btnFocus
}

func (mf *ModalFormModel) GetValue(idx int) string {
	if idx >= 0 && idx < len(mf.fields) {
		return mf.fields[idx].Input.Value()
	}
	return ""
}

func (mf *ModalFormModel) View() string {
	var sb strings.Builder

	sb.WriteString(modalTitleStyle().Render(mf.title))
	sb.WriteString("\n\n")

	for i, f := range mf.fields {
		labelStyle := lipgloss.NewStyle().Foreground(colorSubtle)
		if mf.btnFocus < 0 && i == mf.focused {
			labelStyle = lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
		}
		sb.WriteString(labelStyle.Render(f.Label))
		sb.WriteString("\n")
		sb.WriteString(f.Input.View())
		sb.WriteString("\n\n")
	}

	if len(mf.buttons) > 0 {
		var btns []string
		for i, btn := range mf.buttons {
			if mf.btnFocus == i {
				btns = append(btns, buttonActiveStyle().Render(btn))
			} else {
				btns = append(btns, buttonInactiveStyle.Render(btn))
			}
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, btns...))
	}

	return modalStyle().Width(mf.width).Render(sb.String())
}

// renderModalWithList renders a list-based modal for the given items.
func renderModalWithList(title string, items []ModalListItem, cursor int, footer string, width int) string {
	var sb strings.Builder

	sb.WriteString(modalTitleStyle().Render(title))
	sb.WriteString("\n\n")

	for i, item := range items {
		line := item.Label
		if !item.Selectable {
			sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(line))
		} else if i == cursor {
			sb.WriteString(listCursorStyle().Render("▌ ") + listSelectedStyle().Render(line))
		} else {
			sb.WriteString("  " + line)
		}
		if item.Secondary != "" {
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(colorSubtle).Render("    " + item.Secondary))
		}
		sb.WriteString("\n")
	}

	if footer != "" {
		sb.WriteString("\n")
		sb.WriteString(modalHintStyle.Render(footer))
	}

	return modalStyle().Width(width).Render(sb.String())
}

// renderInputModal renders a simple single-input modal.
func renderInputModal(title string, input textinput.Model, width int) string {
	var sb strings.Builder
	sb.WriteString(modalTitleStyle().Render(title))
	sb.WriteString("\n\n")
	sb.WriteString(input.View())
	sb.WriteString("\n\n")
	sb.WriteString(modalHintStyle.Render("Enter to confirm, Esc to cancel"))
	return modalStyle().Width(width).Render(sb.String())
}

// renderFormModal renders a form modal with fields and buttons.
func renderFormModal(title string, fields []FormField, focusedField int, buttons []string, btnFocus int, width int) string {
	var sb strings.Builder

	sb.WriteString(modalTitleStyle().Render(title))
	sb.WriteString("\n\n")

	for i, f := range fields {
		labelStyle := lipgloss.NewStyle().Foreground(colorSubtle)
		if btnFocus < 0 && i == focusedField {
			labelStyle = lipgloss.NewStyle().Foreground(accentColor()).Bold(true)
		}
		sb.WriteString(labelStyle.Render(f.Label))
		sb.WriteString("\n")
		sb.WriteString(f.Input.View())
		sb.WriteString("\n\n")
	}

	if len(buttons) > 0 {
		var btns []string
		for i, btn := range buttons {
			if btnFocus == i {
				btns = append(btns, buttonActiveStyle().Render(btn))
			} else {
				btns = append(btns, buttonInactiveStyle.Render(btn))
			}
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, btns...))
	}

	return modalStyle().Width(width).Render(sb.String())
}

// fmtEnabled formats enabled/disabled status text.
func fmtEnabled(enabled bool) string {
	if enabled {
		return enabledItemStyle.Render("enabled")
	}
	return disabledItemStyle.Render("disabled")
}

// fmtEnabledIcon returns a styled icon for enabled/disabled state.
func fmtEnabledIcon(enabled bool) string {
	if enabled {
		return fmt.Sprintf("%s %s", enabledItemStyle.Render("*"), "")
	}
	return fmt.Sprintf("%s %s", disabledItemStyle.Render("o"), "")
}
