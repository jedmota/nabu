package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/viewmodel"
)

// MapLocalManager is a modal for managing map local rules
type MapLocalManager struct {
	*tview.List
	viewModel *viewmodel.ViewModel
	app       *tview.Application
	onClose   func()
	onAdd     func()
	onEdit    func(localPath string)
}

// NewMapLocalManager creates a new map local manager
func NewMapLocalManager(vm *viewmodel.ViewModel, app *tview.Application) *MapLocalManager {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Map Local ")
	list.SetTitleAlign(tview.AlignCenter)
	list.ShowSecondaryText(true)
	list.SetHighlightFullLine(true)
	// list.SetBackgroundColor(tcell.ColorDarkBlue)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorGray)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorYellow)

	mm := &MapLocalManager{
		List:      list,
		viewModel: vm,
		app:       app,
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
func (mm *MapLocalManager) SetOnClose(fn func()) {
	mm.onClose = fn
}

// SetOnAdd sets the callback for adding a new mapping
func (mm *MapLocalManager) SetOnAdd(fn func()) {
	mm.onAdd = fn
}

// SetOnEdit sets the callback for editing a mapping file
func (mm *MapLocalManager) SetOnEdit(fn func(localPath string)) {
	mm.onEdit = fn
}

// Refresh updates the list with current mappings
func (mm *MapLocalManager) Refresh() {
	mm.Clear()

	rules := mm.viewModel.GetMapLocalRules()

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
				mm.viewModel.ToggleMapLocalRule(r.ID)
				mm.Refresh()
			})
		}
		mm.AddItem("", "", 0, nil)
		mm.AddItem(fmt.Sprintf("[gray]%d/%d enabled | a: add | e: edit | Space: toggle | d: delete[-]", enabledCount, len(rules)), "", 0, nil)
	}

	mm.SetTitle(fmt.Sprintf(" Map Local [%d] ", len(rules)))
}

// toggleSelected toggles the enabled state of the selected mapping
func (mm *MapLocalManager) toggleSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapLocalRules()

	if idx >= 0 && idx < len(rules) {
		mm.viewModel.ToggleMapLocalRule(rules[idx].ID)
		mm.Refresh()
		mm.SetCurrentItem(idx)
	}
}

// deleteSelected deletes the currently selected mapping
func (mm *MapLocalManager) deleteSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapLocalRules()

	if idx >= 0 && idx < len(rules) {
		mm.viewModel.RemoveMapLocalRule(rules[idx].ID)
		mm.Refresh()

		// Keep selection in bounds
		if idx >= mm.GetItemCount()-2 && idx > 0 {
			mm.SetCurrentItem(idx - 1)
		}
	}
}

// editSelected opens the selected mapping's local file in $EDITOR
func (mm *MapLocalManager) editSelected() {
	idx := mm.GetCurrentItem()
	rules := mm.viewModel.GetMapLocalRules()

	if idx >= 0 && idx < len(rules) {
		localPath := rules[idx].Replacement

		// Expand home directory
		if strings.HasPrefix(localPath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				localPath = home + localPath[1:]
			}
		}

		// Get editor from environment
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vi" // fallback
		}

		// Suspend the TUI and run the editor
		mm.app.Suspend(func() {
			cmd := exec.Command(editor, localPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		})
	}
}
