package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WhichKey shows an overlay with available keybindings
type WhichKey struct {
	*tview.TextView
	keybindings *KeyBindings
	context     Context
	visible     bool
}

// NewWhichKey creates a new WhichKey overlay
func NewWhichKey(keybindings *KeyBindings) *WhichKey {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBorder(true)
	tv.SetTitle(" Keybindings (? to close) ")
	tv.SetTitleAlign(tview.AlignCenter)
	tv.SetScrollable(true)

	wk := &WhichKey{
		TextView:    tv,
		keybindings: keybindings,
		context:     ContextGlobal,
		visible:     false,
	}

	// Add input capture for scrolling
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			row, col := tv.GetScrollOffset()
			if row > 0 {
				tv.ScrollTo(row-1, col)
			}
			return nil
		case tcell.KeyDown:
			row, col := tv.GetScrollOffset()
			tv.ScrollTo(row+1, col)
			return nil
		case tcell.KeyPgUp:
			row, col := tv.GetScrollOffset()
			tv.ScrollTo(row-10, col)
			return nil
		case tcell.KeyPgDn:
			row, col := tv.GetScrollOffset()
			tv.ScrollTo(row+10, col)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				row, col := tv.GetScrollOffset()
				tv.ScrollTo(row+1, col)
				return nil
			case 'k':
				row, col := tv.GetScrollOffset()
				if row > 0 {
					tv.ScrollTo(row-1, col)
				}
				return nil
			}
		}
		return event
	})

	wk.updateContent()
	return wk
}

// SetContext sets the current context for displaying bindings
func (wk *WhichKey) SetContext(context Context) {
	wk.context = context
	wk.updateContent()
}

// Toggle toggles visibility
func (wk *WhichKey) Toggle() {
	wk.visible = !wk.visible
}

// Show shows the overlay
func (wk *WhichKey) Show() {
	wk.visible = true
}

// Hide hides the overlay
func (wk *WhichKey) Hide() {
	wk.visible = false
}

// IsVisible returns whether the overlay is visible
func (wk *WhichKey) IsVisible() bool {
	return wk.visible
}

// updateContent updates the text content
func (wk *WhichKey) updateContent() {
	bindings := wk.keybindings.GetForContext(wk.context)

	// Group by context
	globalBindings := make([]KeyBinding, 0)
	contextBindings := make([]KeyBinding, 0)
	for _, b := range bindings {
		if b.Context == ContextGlobal {
			globalBindings = append(globalBindings, b)
		} else {
			contextBindings = append(contextBindings, b)
		}
	}

	var content string

	// Global section
	if len(globalBindings) > 0 {
		content += "[green::b]Global[-::-]\n"
		for _, b := range globalBindings {
			content += fmt.Sprintf("  [yellow]%-8s[-] %s\n", FormatKey(b), b.Description)
		}
		content += "\n"
	}

	// Context section
	if len(contextBindings) > 0 {
		content += fmt.Sprintf("[green::b]%s[-::-]\n", titleCase(string(wk.context)))
		for _, b := range contextBindings {
			content += fmt.Sprintf("  [yellow]%-8s[-] %s\n", FormatKey(b), b.Description)
		}
		content += "\n"
	}

	content += "[gray]Press ? to close[-]"

	wk.SetText(content)
}

// titleCase converts a string to title case (first letter uppercase)
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}
	return string(r)
}
