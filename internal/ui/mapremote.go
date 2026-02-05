package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/viewmodel"
)

// MapRemoteManager is a modal for managing map remote rules
type MapRemoteManager struct {
	*tview.List
	viewModel *viewmodel.ViewModel
	onClose   func()
	onAdd     func()
	onEdit    func(ruleID int)
}

// NewMapRemoteManager creates a new map remote manager
func NewMapRemoteManager(vm *viewmodel.ViewModel) *MapRemoteManager {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Map Remote ")
	list.SetTitleAlign(tview.AlignCenter)
	list.ShowSecondaryText(true)
	list.SetHighlightFullLine(true)
	// list.SetBackgroundColor(tcell.ColorDarkBlue)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorGray)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorYellow)

	mm := &MapRemoteManager{
		List:      list,
		viewModel: vm,
	}

	// Set up input capture
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if mm.onClose != nil {
				mm.onClose()
			}
			return nil
		case tcell.KeyEnter, tcell.KeyTab:
			mm.toggleSelected()
			return nil
		case tcell.KeyDelete, tcell.KeyBackspace, tcell.KeyBackspace2:
			mm.deleteSelected()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				if mm.onClose != nil {
					mm.onClose()
				}
				return nil
			case ' ':
				mm.toggleSelected()
				return nil
			case 'd', 'x':
				mm.deleteSelected()
				return nil
			case 'a':
				if mm.onAdd != nil {
					mm.onAdd()
				}
				return nil
			case 'e':
				mm.editSelected()
				return nil
			}
		}
		return event
	})

	return mm
}

// SetOnClose sets the callback for when the manager is closed
func (mm *MapRemoteManager) SetOnClose(fn func()) {
	mm.onClose = fn
}

// SetOnAdd sets the callback for adding a new mapping
func (mm *MapRemoteManager) SetOnAdd(fn func()) {
	mm.onAdd = fn
}

// SetOnEdit sets the callback for editing a mapping
func (mm *MapRemoteManager) SetOnEdit(fn func(ruleID int)) {
	mm.onEdit = fn
}

// Refresh updates the list with current mappings
func (mm *MapRemoteManager) Refresh() {
	mm.Clear()

	rules := mm.viewModel.GetMapRemoteRules()

	if len(rules) == 0 {
		mm.AddItem("[gray]No mappings[-]", "", 0, nil)
		mm.AddItem("", "", 0, nil)
		mm.AddItem("[yellow]Press 'a' to add a mapping[-]", "", 0, nil)
		mm.AddItem("[yellow]Press 'Esc' or 'q' to close[-]", "", 0, nil)
	} else {
		enabledCount := 0
		for _, rule := range rules {
			r := rule // capture for closure
			var display string
			if rule.Enabled {
				display = fmt.Sprintf("[green]✓[-] %s", rule.Pattern)
				enabledCount++
			} else {
				display = fmt.Sprintf("[gray]○ %s[-]", rule.Pattern)
			}
			secondary := fmt.Sprintf("  → %s", rule.Replacement)
			mm.AddItem(display, secondary, 0, func() {
				mm.viewModel.ToggleMapRemoteRule(r.ID)
				mm.Refresh()
			})
		}
		mm.AddItem("", "", 0, nil)
		mm.AddItem(fmt.Sprintf("[gray]%d/%d enabled | a: add | e: edit | Space: toggle | d: delete[-]", enabledCount, len(rules)), "", 0, nil)
	}

	mm.SetTitle(fmt.Sprintf(" Map Remote [%d] ", len(rules)))
}

// toggleSelected toggles the enabled state of the selected mapping
func (mm *MapRemoteManager) toggleSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapRemoteRules()

	if idx >= 0 && idx < len(rules) {
		mm.viewModel.ToggleMapRemoteRule(rules[idx].ID)
		mm.Refresh()
		mm.SetCurrentItem(idx)
	}
}

// deleteSelected deletes the currently selected mapping
func (mm *MapRemoteManager) deleteSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapRemoteRules()

	if idx >= 0 && idx < len(rules) {
		mm.viewModel.RemoveMapRemoteRule(rules[idx].ID)
		mm.Refresh()

		// Keep selection in bounds
		if idx >= mm.GetItemCount()-2 && idx > 0 {
			mm.SetCurrentItem(idx - 1)
		}
	}
}

// editSelected edits the currently selected mapping
func (mm *MapRemoteManager) editSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapRemoteRules()

	if idx >= 0 && idx < len(rules) && mm.onEdit != nil {
		mm.onEdit(rules[idx].ID)
	}
}
