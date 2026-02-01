package ui

import (
	"github.com/rivo/tview"
)

// Layout manages the UI layout
type Layout struct {
	grid          *tview.Grid
	requestsPanel *RequestsPanel
	detailPanel   *DetailPanel
	statusBar     *tview.TextView
	expanded      bool
	focusedPanel  int // 0 = requests, 1 = detail
}

// NewLayout creates a new layout
func NewLayout(requestsPanel *RequestsPanel, detailPanel *DetailPanel) *Layout {
	l := &Layout{
		requestsPanel: requestsPanel,
		detailPanel:   detailPanel,
		focusedPanel:  0,
	}

	// Create status bar
	l.statusBar = tview.NewTextView()
	l.statusBar.SetDynamicColors(true)
	l.statusBar.SetText(" [yellow]Proxy TUI[-] | [green]Tab[-]: switch panel | [green]?[-]: help | [green]q[-]: quit")

	// Create grid layout
	l.grid = tview.NewGrid()
	l.setupNormalLayout()

	return l
}

// setupNormalLayout sets up the default two-panel layout
func (l *Layout) setupNormalLayout() {
	l.grid.Clear()
	l.grid.SetRows(0, 1)    // Main content, status bar
	l.grid.SetColumns(-1, -1) // Two equal columns

	l.grid.AddItem(l.requestsPanel, 0, 0, 1, 1, 0, 0, true)
	l.grid.AddItem(l.detailPanel, 0, 1, 1, 1, 0, 0, false)
	l.grid.AddItem(l.statusBar, 1, 0, 1, 2, 0, 0, false)

	l.expanded = false
}

// setupExpandedLayout sets up a single-panel expanded layout
func (l *Layout) setupExpandedLayout() {
	l.grid.Clear()
	l.grid.SetRows(0, 1)    // Main content, status bar
	l.grid.SetColumns(0)    // Single column

	if l.focusedPanel == 0 {
		l.grid.AddItem(l.requestsPanel, 0, 0, 1, 1, 0, 0, true)
	} else {
		l.grid.AddItem(l.detailPanel, 0, 0, 1, 1, 0, 0, true)
	}
	l.grid.AddItem(l.statusBar, 1, 0, 1, 1, 0, 0, false)

	l.expanded = true
}

// ToggleExpand toggles between normal and expanded layout
func (l *Layout) ToggleExpand() {
	if l.expanded {
		l.setupNormalLayout()
	} else {
		l.setupExpandedLayout()
	}
}

// ToggleFocus toggles focus between panels
func (l *Layout) ToggleFocus(app *tview.Application) {
	l.focusedPanel = (l.focusedPanel + 1) % 2

	if l.focusedPanel == 0 {
		app.SetFocus(l.requestsPanel)
	} else {
		app.SetFocus(l.detailPanel)
	}

	l.updateStatusBar()
}

// SetFocus sets focus to a specific panel
func (l *Layout) SetFocus(panel int, app *tview.Application) {
	l.focusedPanel = panel
	if panel == 0 {
		app.SetFocus(l.requestsPanel)
	} else {
		app.SetFocus(l.detailPanel)
	}
	l.updateStatusBar()
}

// GetFocusedPanel returns the currently focused panel index
func (l *Layout) GetFocusedPanel() int {
	return l.focusedPanel
}

// updateStatusBar updates the status bar text
func (l *Layout) updateStatusBar() {
	panelName := "Requests"
	if l.focusedPanel == 1 {
		panelName = "Detail"
	}
	l.statusBar.SetText(" [yellow]Proxy TUI[-] | [cyan]" + panelName + "[-] | [green]Tab[-]: switch | [green]?[-]: help | [green]q[-]: quit")
}

// SetStatus sets a custom status message
func (l *Layout) SetStatus(msg string) {
	l.statusBar.SetText(" " + msg)
}

// Grid returns the root grid
func (l *Layout) Grid() *tview.Grid {
	return l.grid
}

// IsExpanded returns whether the layout is expanded
func (l *Layout) IsExpanded() bool {
	return l.expanded
}
