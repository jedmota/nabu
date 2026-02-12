package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// AlertManager is a modal for managing alert rules.
type AlertManager struct {
	*tview.List
	viewModel *viewmodel.ViewModel
	onClose   func()
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(vm *viewmodel.ViewModel) *AlertManager {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Alert Settings ")
	list.SetTitleAlign(tview.AlignCenter)
	list.ShowSecondaryText(true)
	list.SetHighlightFullLine(true)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorYellow)
	list.SetSecondaryTextColor(tcell.ColorGray)

	am := &AlertManager{
		List:      list,
		viewModel: vm,
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if am.onClose != nil {
				am.onClose()
			}
			return nil
		case tcell.KeyEnter, tcell.KeyTab:
			am.toggleSelected()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				if am.onClose != nil {
					am.onClose()
				}
				return nil
			case ' ':
				am.toggleSelected()
				return nil
			}
		}
		return event
	})

	return am
}

// SetOnClose sets the callback for closing the alert manager.
func (am *AlertManager) SetOnClose(fn func()) {
	am.onClose = fn
}

// Refresh reloads alert rules into the list.
func (am *AlertManager) Refresh() {
	am.Clear()
	rules := am.viewModel.GetAlertRules()

	for _, rule := range rules {
		label := formatAlertRule(rule)
		status := "[red]disabled[-]"
		if rule.Enabled {
			status = "[green]enabled[-]"
		}
		am.AddItem(label, "  "+status+"  [gray](Enter/Space to toggle)[-]", 0, nil)
	}

	if len(rules) == 0 {
		am.AddItem("[gray]No alert rules configured[-]", "", 0, nil)
	}
}

func (am *AlertManager) toggleSelected() {
	index := am.GetCurrentItem()
	rules := am.viewModel.GetAlertRules()
	if index < 0 || index >= len(rules) {
		return
	}
	am.viewModel.ToggleAlertRule(index)
	am.Refresh()
}

func formatAlertRule(rule model.AlertRule) string {
	switch rule.Type {
	case model.AlertStatusCode:
		return fmt.Sprintf("Status %dxx", rule.Value/100)
	case model.AlertLatency:
		if rule.Value >= 1000 {
			return fmt.Sprintf("Latency > %.1fs", float64(rule.Value)/1000)
		}
		return fmt.Sprintf("Latency > %dms", rule.Value)
	default:
		return string(rule.Type)
	}
}
