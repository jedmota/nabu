package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/model"
	"proxy-tui/internal/viewmodel"
)

// App is the main TUI application
type App struct {
	tviewApp      *tview.Application
	viewModel     *viewmodel.ViewModel
	layout        *Layout
	requestsPanel *RequestsPanel
	detailPanel   *DetailPanel
	whichKey      *WhichKey
	keybindings   *KeyBindings
	pages         *tview.Pages
	searchInput   *tview.InputField
	searching     bool
}

// NewApp creates a new TUI application
func NewApp(vm *viewmodel.ViewModel) *App {
	app := &App{
		tviewApp:    tview.NewApplication(),
		viewModel:   vm,
		keybindings: NewKeyBindings(),
	}

	// Create UI components
	app.requestsPanel = NewRequestsPanel(vm)
	app.detailPanel = NewDetailPanel(vm)
	app.whichKey = NewWhichKey(app.keybindings)
	app.layout = NewLayout(app.requestsPanel, app.detailPanel)

	// Create search input
	app.searchInput = tview.NewInputField()
	app.searchInput.SetLabel("Search: ")
	app.searchInput.SetFieldWidth(40)
	app.searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyEsc {
			app.closeSearch()
		}
	})
	app.searchInput.SetChangedFunc(func(text string) {
		vm.SetSearchQuery(text)
		app.requestsPanel.Refresh()
	})

	// Setup pages for overlays
	app.pages = tview.NewPages()
	app.pages.AddPage("main", app.layout.Grid(), true, true)

	// Create search modal
	searchFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.searchInput, 3, 0, true).
			AddItem(nil, 0, 1, false), 50, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("search", searchFlex, true, false)

	// Create whichkey modal (centered overlay)
	whichKeyFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.whichKey, 20, 0, false).
			AddItem(nil, 0, 1, false), 60, 0, false).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("whichkey", whichKeyFlex, true, false)

	// Set up selection callback
	app.requestsPanel.SetOnSelect(func(flow *model.Flow) {
		app.detailPanel.SetFlow(flow)
	})

	// Set up global input handler
	app.tviewApp.SetInputCapture(app.handleInput)

	// Start listening for updates
	go app.listenForUpdates()

	return app
}

// Run starts the TUI application
func (app *App) Run() error {
	app.tviewApp.SetRoot(app.pages, true)
	app.tviewApp.EnableMouse(true)

	// Initial refresh
	app.requestsPanel.Refresh()

	return app.tviewApp.Run()
}

// Stop stops the TUI application
func (app *App) Stop() {
	app.tviewApp.Stop()
}

// listenForUpdates listens for ViewModel updates and refreshes the UI
func (app *App) listenForUpdates() {
	for range app.viewModel.Updates() {
		app.tviewApp.QueueUpdateDraw(func() {
			app.requestsPanel.Refresh()
			// Update detail if we have a selection
			if flow := app.viewModel.GetSelectedFlow(); flow != nil {
				app.detailPanel.SetFlow(flow)
			}
		})
	}
}

// handleInput handles global keyboard input
func (app *App) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// If searching, let the search input handle it
	if app.searching {
		return event
	}

	// Determine current context
	context := ContextList
	if app.layout.GetFocusedPanel() == 1 {
		context = ContextDetail
	}

	// Look up keybinding
	binding := app.keybindings.Lookup(event.Key(), event.Rune(), context)
	if binding == nil {
		return event
	}

	// Update whichkey context
	app.whichKey.SetContext(context)

	// Handle action
	switch binding.Action {
	case ActionQuit:
		app.Stop()
		return nil

	case ActionToggleFocus:
		app.layout.ToggleFocus(app.tviewApp)
		return nil

	case ActionExpandPanel:
		app.layout.ToggleExpand()
		return nil

	case ActionToggleHelp:
		app.toggleWhichKey()
		return nil

	case ActionNextItem:
		if context == ContextList {
			app.requestsPanel.MoveDown()
		} else {
			app.detailPanel.ScrollDown()
		}
		return nil

	case ActionPrevItem:
		if context == ContextList {
			app.requestsPanel.MoveUp()
		} else {
			app.detailPanel.ScrollUp()
		}
		return nil

	case ActionSelectItem:
		// Selection is handled by table
		return event

	case ActionRefresh:
		app.viewModel.Refresh()
		app.requestsPanel.Refresh()
		return nil

	case ActionSearch:
		app.openSearch()
		return nil

	case ActionFilterAll:
		app.viewModel.SetFilterType(model.FilterAll)
		app.requestsPanel.Refresh()
		app.layout.SetStatus("[green]Filter: All[-]")
		return nil

	case ActionFilterWhite:
		app.viewModel.SetFilterType(model.FilterWhitelist)
		app.requestsPanel.Refresh()
		app.layout.SetStatus("[green]Filter: Whitelist[-]")
		return nil

	case ActionClearFlows:
		app.viewModel.ClearFlows()
		app.detailPanel.Clear()
		app.requestsPanel.Refresh()
		return nil

	case ActionToggleRaw:
		app.detailPanel.ToggleRawMode()
		return nil

	case ActionScrollUp:
		app.detailPanel.ScrollUp()
		return nil

	case ActionScrollDown:
		app.detailPanel.ScrollDown()
		return nil

	case ActionPageUp:
		app.detailPanel.PageUp()
		return nil

	case ActionPageDown:
		app.detailPanel.PageDown()
		return nil
	}

	return event
}

// toggleWhichKey toggles the WhichKey overlay
func (app *App) toggleWhichKey() {
	app.whichKey.Toggle()
	if app.whichKey.IsVisible() {
		app.pages.ShowPage("whichkey")
	} else {
		app.pages.HidePage("whichkey")
	}
}

// openSearch opens the search input
func (app *App) openSearch() {
	app.searching = true
	app.searchInput.SetText(app.viewModel.GetFilter().SearchQuery)
	app.pages.ShowPage("search")
	app.tviewApp.SetFocus(app.searchInput)
}

// closeSearch closes the search input
func (app *App) closeSearch() {
	app.searching = false
	app.pages.HidePage("search")
	app.layout.SetFocus(0, app.tviewApp)
}

// ShowMessage shows a temporary message in the status bar
func (app *App) ShowMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	app.layout.SetStatus(msg)
}
