package ui

import (
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
)

// Layout manages the UI layout
type Layout struct {
	grid          *tview.Grid
	requestsPanel *RequestsPanel
	detailPanel   *DetailPanel
	filterBar     *tview.TextView
	statusBar     *tview.TextView
	addressBar    *tview.TextView
	statusFlex    *tview.Flex
	expanded      bool
	focusedPanel  int              // 0 = requests, 1 = detail
	activeFilter  model.FilterType // current active filter
	customPattern string           // custom filter pattern
}

// NewLayout creates a new layout
func NewLayout(requestsPanel *RequestsPanel, detailPanel *DetailPanel) *Layout {
	l := &Layout{
		requestsPanel: requestsPanel,
		detailPanel:   detailPanel,
		focusedPanel:  0,
		activeFilter:  model.FilterAll,
	}

	// Create filter bar at the top
	l.filterBar = tview.NewTextView()
	l.filterBar.SetDynamicColors(true)
	l.filterBar.SetTextAlign(tview.AlignCenter)
	l.updateFilterBar()

	// Create status bar (left side)
	l.statusBar = tview.NewTextView()
	l.statusBar.SetDynamicColors(true)
	l.statusBar.SetText(" [cyan]Requests[-] [gray]|[-] [green]l[-]:local [green]L[-]:local mgr [green]r[-]:remote [green]R[-]:remote mgr [green]w[-]:whitelist [green]c[-]:clear [green]?[-]:help")

	// Create address bar (right side)
	l.addressBar = tview.NewTextView()
	l.addressBar.SetDynamicColors(true)
	l.addressBar.SetTextAlign(tview.AlignRight)

	// Create flex container for status bar
	l.statusFlex = tview.NewFlex().
		AddItem(l.statusBar, 0, 1, false).
		AddItem(l.addressBar, 25, 0, false)

	// Create grid layout
	l.grid = tview.NewGrid()
	l.setupNormalLayout()

	// Set initial panel border colors
	l.updatePanelBorders()

	return l
}

// setupNormalLayout sets up the default two-panel layout
func (l *Layout) setupNormalLayout() {
	l.grid.Clear()
	l.grid.SetRows(1, 0, 1)    // Filter bar, main content, status bar
	l.grid.SetColumns(-13, -7) // 55% requests, 45% detail

	l.grid.AddItem(l.filterBar, 0, 0, 1, 2, 0, 0, false)
	l.grid.AddItem(l.requestsPanel, 1, 0, 1, 1, 0, 0, true)
	l.grid.AddItem(l.detailPanel, 1, 1, 1, 1, 0, 0, false)
	l.grid.AddItem(l.statusFlex, 2, 0, 1, 2, 0, 0, false)

	l.expanded = false
}

// setupExpandedLayout sets up a single-panel expanded layout
func (l *Layout) setupExpandedLayout() {
	l.grid.Clear()
	l.grid.SetRows(1, 0, 1) // Filter bar, main content, status bar
	l.grid.SetColumns(0)    // Single column

	l.grid.AddItem(l.filterBar, 0, 0, 1, 1, 0, 0, false)
	if l.focusedPanel == 0 {
		l.grid.AddItem(l.requestsPanel, 1, 0, 1, 1, 0, 0, true)
	} else {
		l.grid.AddItem(l.detailPanel, 1, 0, 1, 1, 0, 0, true)
	}
	l.grid.AddItem(l.statusFlex, 2, 0, 1, 1, 0, 0, false)

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

	l.updatePanelBorders()
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
	l.updatePanelBorders()
	l.updateStatusBar()
}

// updatePanelBorders updates panel border colors based on focus
func (l *Layout) updatePanelBorders() {
	l.requestsPanel.SetFocused(l.focusedPanel == 0)
	l.detailPanel.SetFocused(l.focusedPanel == 1)
}

// GetFocusedPanel returns the currently focused panel index
func (l *Layout) GetFocusedPanel() int {
	return l.focusedPanel
}

// updateStatusBar updates the status bar text with context-specific keys
func (l *Layout) updateStatusBar() {
	if l.focusedPanel == 0 {
		// Requests panel context
		l.statusBar.SetText(" [cyan]Requests[-] [gray]|[-] [green]l[-]:local [green]L[-]:local mgr [green]r[-]:remote [green]R[-]:remote mgr [green]w[-]:whitelist [green]c[-]:clear [green]?[-]:help")
	} else {
		// Detail panel context
		l.statusBar.SetText(" [cyan]Detail[-] [gray]|[-] [green]j/k[-]:scroll [green]T[-]:raw [green]l[-]:local [green]r[-]:remote [green]w[-]:whitelist [green]?[-]:help")
	}
}

// SetStatus sets a custom status message
func (l *Layout) SetStatus(msg string) {
	l.statusBar.SetText(" " + msg)
}

// SetAddress sets the proxy address displayed on the right
func (l *Layout) SetAddress(addr string) {
	l.addressBar.SetText("[gray]" + addr + " [-]")
}

// Grid returns the root grid
func (l *Layout) Grid() *tview.Grid {
	return l.grid
}

// IsExpanded returns whether the layout is expanded
func (l *Layout) IsExpanded() bool {
	return l.expanded
}

// SetFilter sets the active filter and updates the filter bar
func (l *Layout) SetFilter(filterType model.FilterType) {
	l.activeFilter = filterType
	l.updateFilterBar()
}

// updateFilterBar updates the filter bar to show the active filter
func (l *Layout) updateFilterBar() {
	allStyle := "[gray]"
	whitelistStyle := "[gray]"
	starStyle := "[gray]"
	customStyle := "[gray]"

	switch l.activeFilter {
	case model.FilterAll:
		allStyle = "[white:black]"
	case model.FilterWhitelist:
		whitelistStyle = "[white:black]"
	case model.FilterStarred:
		starStyle = "[white:black]"
	case model.FilterCustom:
		customStyle = "[white:black]"
	}

	customText := " /:Custom "
	if l.customPattern != "" {
		customText = " /:" + l.customPattern + " "
	}

	l.filterBar.SetText(allStyle + " 1:All [-] " + whitelistStyle + " 2:Whitelist [-] " + starStyle + " 3:Starred [-] " + customStyle + customText + "[-]")
}

// SetCustomPattern sets the custom filter pattern
func (l *Layout) SetCustomPattern(pattern string) {
	l.customPattern = pattern
	l.updateFilterBar()
}
