package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/viewmodel"
)

// WhitelistManager is a modal for managing whitelist patterns
type WhitelistManager struct {
	*tview.List
	viewModel *viewmodel.ViewModel
	onClose   func()
}

// NewWhitelistManager creates a new whitelist manager
func NewWhitelistManager(vm *viewmodel.ViewModel) *WhitelistManager {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Whitelist Manager ")
	list.SetTitleAlign(tview.AlignCenter)
	list.ShowSecondaryText(false)
	list.SetHighlightFullLine(true)
	// list.SetBackgroundColor(tcell.ColorDarkBlue)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorYellow)

	wm := &WhitelistManager{
		List:      list,
		viewModel: vm,
	}

	// Set up input capture for toggle, delete and close
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if wm.onClose != nil {
				wm.onClose()
			}
			return nil
		case tcell.KeyEnter, tcell.KeyTab:
			wm.toggleSelected()
			return nil
		case tcell.KeyDelete, tcell.KeyBackspace, tcell.KeyBackspace2:
			wm.deleteSelected()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				if wm.onClose != nil {
					wm.onClose()
				}
				return nil
			case ' ':
				wm.toggleSelected()
				return nil
			case 'd', 'x':
				wm.deleteSelected()
				return nil
			}
		}
		return event
	})

	return wm
}

// SetOnClose sets the callback for when the manager is closed
func (wm *WhitelistManager) SetOnClose(fn func()) {
	wm.onClose = fn
}

// Refresh updates the list with current patterns
func (wm *WhitelistManager) Refresh() {
	wm.Clear()

	patterns := wm.viewModel.GetWhitelistPatterns()

	if len(patterns) == 0 {
		wm.AddItem("[gray]No patterns[-]", "", 0, nil)
		wm.AddItem("", "", 0, nil)
		wm.AddItem("[yellow]Press 'w' to add patterns[-]", "", 0, nil)
		wm.AddItem("[yellow]Press 'Esc' or 'q' to close[-]", "", 0, nil)
	} else {
		enabledCount := 0
		for _, hp := range patterns {
			p := hp.Pattern // capture for closure
			var display string
			if hp.Enabled {
				display = fmt.Sprintf("[green]✓[-] %s", hp.Pattern)
				enabledCount++
			} else {
				display = fmt.Sprintf("[gray]○ %s[-]", hp.Pattern)
			}
			wm.AddItem(display, "", 0, func() {
				wm.viewModel.ToggleWhitelistPattern(p)
				wm.Refresh()
			})
		}
		wm.AddItem("", "", 0, nil)
		wm.AddItem(fmt.Sprintf("[gray]%d/%d enabled | Space: toggle | d: delete | Esc: close[-]", enabledCount, len(patterns)), "", 0, nil)
	}

	wm.SetTitle(fmt.Sprintf(" Whitelist Manager [%d] ", len(patterns)))
}

// toggleSelected toggles the enabled state of the selected pattern
func (wm *WhitelistManager) toggleSelected() {
	idx := wm.GetCurrentItem()
	patterns := wm.viewModel.GetWhitelistPatterns()

	if idx >= 0 && idx < len(patterns) {
		wm.viewModel.ToggleWhitelistPattern(patterns[idx].Pattern)
		wm.Refresh()
		wm.SetCurrentItem(idx)
	}
}

// deleteSelected deletes the currently selected pattern
func (wm *WhitelistManager) deleteSelected() {
	idx := wm.GetCurrentItem()
	patterns := wm.viewModel.GetWhitelistPatterns()

	if idx >= 0 && idx < len(patterns) {
		wm.viewModel.RemoveWhitelistPattern(patterns[idx].Pattern)
		wm.Refresh()

		// Keep selection in bounds
		if idx >= wm.GetItemCount()-2 && idx > 0 {
			wm.SetCurrentItem(idx - 1)
		}
	}
}
