package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"proxy-tui/internal/config"
	"proxy-tui/internal/model"
	"proxy-tui/internal/util"
	"proxy-tui/internal/viewmodel"
)

// PopupState represents which popup (if any) is currently active.
type PopupState int

const (
	PopupNone PopupState = iota
	PopupSearch
	PopupWhitelistInput
	PopupMapLocalPattern
	PopupWhitelistManager
	PopupMapLocalManager
	PopupMapLocalForm
	PopupMapRemoteManager
	PopupMapRemoteForm
	PopupAlertManager
	PopupImportHAR
)

// App is the main TUI application
type App struct {
	tviewApp             *tview.Application
	viewModel            *viewmodel.ViewModel
	layout               *Layout
	requestsPanel        *RequestsPanel
	detailPanel          *DetailPanel
	whichKey             *WhichKey
	keybindings          *KeyBindings
	pages                *tview.Pages
	searchInput          *tview.InputField
	whitelistInput       *tview.InputField
	mapLocalPatternInput *tview.TextArea
	whitelistManager     *WhitelistManager
	mapLocalManager      *MapLocalManager
	mapLocalForm         *tview.Form
	mapRemoteManager     *MapRemoteManager
	mapRemoteForm        *tview.Form
	alertManager         *AlertManager
	filePicker           *FilePicker
	activePopup          PopupState
	lastKeyRune          rune
	lastKeyTime          time.Time
}

// NewApp creates a new TUI application
func NewApp(vm *viewmodel.ViewModel) *App {
	// Set single-line border characters (both normal and focused)
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'
	tview.Borders.TopLeft = '┌'
	tview.Borders.TopRight = '┐'
	tview.Borders.BottomLeft = '└'
	tview.Borders.BottomRight = '┘'
	tview.Borders.LeftT = '├'
	tview.Borders.RightT = '┤'
	tview.Borders.TopT = '┬'
	tview.Borders.BottomT = '┴'
	tview.Borders.Cross = '┼'
	tview.Borders.HorizontalFocus = '─'
	tview.Borders.VerticalFocus = '│'
	tview.Borders.TopLeftFocus = '┌'
	tview.Borders.TopRightFocus = '┐'
	tview.Borders.BottomLeftFocus = '└'
	tview.Borders.BottomRightFocus = '┘'

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
	app.searchInput.SetLabel("Filter: ")
	app.searchInput.SetFieldWidth(40)
	app.searchInput.SetFieldBackgroundColor(tcell.ColorDefault)
	app.searchInput.SetFieldTextColor(tcell.ColorWhite)
	app.searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyEsc {
			app.closeSearch()
		}
	})
	app.searchInput.SetChangedFunc(func(text string) {
		vm.SetSearchQuery(text)
		vm.SetFilterType(model.FilterCustom)
		app.layout.SetCustomPattern(text)
		app.layout.SetFilter(model.FilterCustom)
		app.requestsPanel.Refresh()
	})

	// Create whitelist input
	app.whitelistInput = tview.NewInputField()
	app.whitelistInput.SetBorder(true)
	app.whitelistInput.SetTitle(" Add Pattern ")
	app.whitelistInput.SetTitleAlign(tview.AlignCenter)
	app.whitelistInput.SetLabel(" Pattern (e.g., *.example.com): ")
	app.whitelistInput.SetFieldWidth(0)
	app.whitelistInput.SetFieldBackgroundColor(tcell.ColorDefault)
	app.whitelistInput.SetFieldTextColor(tcell.ColorWhite)
	app.whitelistInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			pattern := app.whitelistInput.GetText()
			if pattern != "" {
				vm.AddWhitelistPattern(pattern)
				app.layout.SetStatus(fmt.Sprintf("[green]Added pattern: %s[-]", pattern))
			}
		}
		app.closeWhitelistInput()
	})

	// Create map local pattern input
	app.mapLocalPatternInput = tview.NewTextArea()
	app.mapLocalPatternInput.SetTitle(" URL Pattern (Ctrl+S to save, Esc to cancel) ")
	app.mapLocalPatternInput.SetBorder(true)
	app.mapLocalPatternInput.SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDefault))
	app.mapLocalPatternInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.closeMapLocalPatternInput()
			return nil
		}
		// Ctrl+S to save
		if event.Key() == tcell.KeyCtrlS {
			pattern := strings.TrimSpace(app.mapLocalPatternInput.GetText())
			if pattern != "" {
				app.createMapLocalWithPattern(pattern)
			}
			app.closeMapLocalPatternInput()
			return nil
		}
		return event
	})

	// Create whitelist manager
	app.whitelistManager = NewWhitelistManager(vm)
	app.whitelistManager.SetOnClose(func() {
		app.closeWhitelistManager()
	})
	app.whitelistManager.SetOnEdit(func(pattern string) {
		app.editWhitelistPattern(pattern)
	})
	app.whitelistManager.SetOnAdd(func() {
		app.addWhitelistFromManager()
	})

	// Create map local manager
	app.mapLocalManager = NewMapLocalManager(vm, app.tviewApp)
	app.mapLocalManager.SetOnClose(func() {
		app.closeMapLocalManager()
	})
	app.mapLocalManager.SetOnAdd(func() {
		app.openMapLocalForm()
	})

	// Create map local form
	app.createMapLocalForm()

	// Create map remote manager
	app.mapRemoteManager = NewMapRemoteManager(vm)
	app.mapRemoteManager.SetOnClose(func() {
		app.closeMapRemoteManager()
	})
	app.mapRemoteManager.SetOnAdd(func() {
		app.openMapRemoteForm()
	})
	app.mapRemoteManager.SetOnEdit(func(ruleID int) {
		app.editMapRemoteRule(ruleID)
	})

	// Create map remote form
	app.createMapRemoteForm()

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

	// Create whitelist input modal
	whitelistFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.whitelistInput, 3, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("whitelist-input", whitelistFlex, true, false)

	// Create map local pattern input modal
	mapLocalPatternFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.mapLocalPatternInput, 8, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("maplocal-pattern", mapLocalPatternFlex, true, false)

	// Create whitelist manager modal
	whitelistManagerFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.whitelistManager, 30, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("whitelist-manager", whitelistManagerFlex, true, false)

	// Create map local manager modal
	mapLocalManagerFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.mapLocalManager, 30, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("maplocal-manager", mapLocalManagerFlex, true, false)

	// Create map local form modal
	mapLocalFormFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.mapLocalForm, 18, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("maplocal-form", mapLocalFormFlex, true, false)

	// Create map remote manager modal
	mapRemoteManagerFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.mapRemoteManager, 30, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("mapremote-manager", mapRemoteManagerFlex, true, false)

	// Create map remote form modal
	mapRemoteFormFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.mapRemoteForm, 18, 0, true).
			AddItem(nil, 0, 1, false), 90, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("mapremote-form", mapRemoteFormFlex, true, false)

	// Create alert manager
	app.alertManager = NewAlertManager(vm)
	app.alertManager.SetOnClose(func() {
		app.closeAlertManager()
	})

	alertManagerFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.alertManager, 20, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("alert-manager", alertManagerFlex, true, false)

	// Create file picker for HAR import
	app.filePicker = NewFilePicker()
	app.filePicker.SetOnSelect(func(path string) {
		app.closeImportInput()
		app.doImportHAR(path)
	})
	app.filePicker.SetOnClose(func() {
		app.closeImportInput()
	})
	app.pages.AddPage("import-har", app.filePicker, true, false)

	// Create whichkey modal (centered overlay)
	whichKeyFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.whichKey, 30, 0, false).
			AddItem(nil, 0, 1, false), 70, 0, false).
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

	// Prevent mouse clicks outside popups from stealing focus
	app.tviewApp.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if app.isPopupOpen() && (action == tview.MouseLeftClick || action == tview.MouseLeftDoubleClick) {
			popup := app.getCurrentPopupPrimitive()
			if popup != nil {
				x, y := event.Position()
				px, py, pw, ph := popup.GetRect()
				// Check if click is within popup bounds
				if x >= px && x < px+pw && y >= py && y < py+ph {
					return event, action
				}
			}
			// Click is outside popup, block it completely
			return nil, 0
		}
		return event, action
	})

	// Set proxy address in status bar
	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, app.viewModel.Port())
	if app.viewModel.IsSecondary() {
		addr = "[yellow]IPC[-] " + addr
	}
	app.layout.SetAddress(addr)

	// Initial refresh
	app.requestsPanel.Refresh()

	return app.tviewApp.Run()
}

// getLocalIP returns the preferred outbound IP of this machine
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// Stop stops the TUI application
func (app *App) Stop() {
	app.tviewApp.Stop()
}

// isPopupOpen returns true if any popup/modal is currently open
func (app *App) isPopupOpen() bool {
	return app.activePopup != PopupNone || app.whichKey.IsVisible()
}

// getCurrentPopupPrimitive returns the currently active popup primitive
func (app *App) getCurrentPopupPrimitive() tview.Primitive {
	switch app.activePopup {
	case PopupSearch:
		return app.searchInput
	case PopupWhitelistInput:
		return app.whitelistInput
	case PopupMapLocalPattern:
		return app.mapLocalPatternInput
	case PopupWhitelistManager:
		return app.whitelistManager
	case PopupMapLocalManager:
		return app.mapLocalManager
	case PopupMapLocalForm:
		return app.mapLocalForm
	case PopupMapRemoteManager:
		return app.mapRemoteManager
	case PopupMapRemoteForm:
		return app.mapRemoteForm
	case PopupAlertManager:
		return app.alertManager
	case PopupImportHAR:
		return app.filePicker
	}
	if app.whichKey.IsVisible() {
		return app.whichKey
	}
	return nil
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
	// If whichkey is visible, only handle '?' to close it, otherwise close on any key
	if app.whichKey.IsVisible() {
		if event.Key() == tcell.KeyRune && event.Rune() == '?' {
			app.toggleWhichKey()
		} else if event.Key() == tcell.KeyEsc {
			app.whichKey.Hide()
			app.pages.HidePage("whichkey")
		} else {
			// Close whichkey on any other key and don't process the key
			app.whichKey.Hide()
			app.pages.HidePage("whichkey")
		}
		return nil
	}

	// If in input mode, let the input handle it
	if app.activePopup != PopupNone {
		return event
	}

	// Determine current context
	context := ContextList
	if app.layout.GetFocusedPanel() == 1 {
		context = ContextDetail
	}

	// Handle vim-style chord: gg (go to top with stay on top)
	if event.Key() == tcell.KeyRune && event.Rune() == 'g' {
		if app.lastKeyRune == 'g' && time.Since(app.lastKeyTime) < 500*time.Millisecond {
			app.lastKeyRune = 0
			if context == ContextList {
				app.requestsPanel.GoToTop()
			}
			return nil
		}
		app.lastKeyRune = 'g'
		app.lastKeyTime = time.Now()
		return nil
	}

	// Handle G (go to bottom)
	if event.Key() == tcell.KeyRune && event.Rune() == 'G' {
		if context == ContextList {
			app.requestsPanel.GoToBottom()
		}
		return nil
	}

	// Reset chord tracking for other keys
	app.lastKeyRune = 0

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
		app.layout.SetFilter(model.FilterAll)
		app.requestsPanel.Refresh()
		return nil

	case ActionFilterWhite:
		app.viewModel.SetFilterType(model.FilterWhitelist)
		app.layout.SetFilter(model.FilterWhitelist)
		app.requestsPanel.Refresh()
		return nil

	case ActionClearFlows:
		app.viewModel.ClearFlows()
		app.detailPanel.Clear()
		app.requestsPanel.Refresh()
		return nil

	case ActionAddWhitelist:
		app.openWhitelistInput()
		return nil

	case ActionShowWhitelist:
		app.showWhitelistManager()
		return nil

	case ActionClearWhitelist:
		app.viewModel.ClearWhitelist()
		app.requestsPanel.Refresh()
		app.layout.SetStatus("[yellow]Whitelist cleared[-]")
		return nil

	case ActionMapLocal:
		app.showMapLocalManager()
		return nil

	case ActionQuickMapLocal:
		app.openMapLocalPatternInput()
		return nil

	case ActionMapRemote:
		app.showMapRemoteManager()
		return nil

	case ActionAddMapRemote:
		app.openMapRemoteForm()
		return nil

	case ActionReplay:
		app.replaySelectedFlow()
		return nil

	case ActionCopyCURL:
		app.copyCURL()
		return nil

	case ActionExportHAR:
		app.exportHAR(false)
		return nil

	case ActionExportAllHAR:
		app.exportHAR(true)
		return nil

	case ActionAlerts:
		app.showAlertManager()
		return nil

	case ActionImportHAR:
		app.openImportInput()
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
	app.activePopup = PopupSearch
	app.searchInput.SetText(app.viewModel.GetFilter().SearchQuery)
	app.pages.ShowPage("search")
	app.tviewApp.SetFocus(app.searchInput)
}

// closeSearch closes the search input
func (app *App) closeSearch() {
	app.activePopup = PopupNone
	app.pages.HidePage("search")
	app.layout.SetFocus(0, app.tviewApp)
}

// openWhitelistInput opens the whitelist pattern input
func (app *App) openWhitelistInput() {
	app.activePopup = PopupWhitelistInput

	// Pre-fill with selected flow's host if available
	if flow := app.viewModel.GetSelectedFlow(); flow != nil && flow.Request != nil {
		host := flow.Request.Host
		// Remove port if present
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		app.whitelistInput.SetText(host)
	} else {
		app.whitelistInput.SetText("")
	}

	app.pages.ShowPage("whitelist-input")
	app.tviewApp.SetFocus(app.whitelistInput)
}

// closeWhitelistInput closes the whitelist input
func (app *App) closeWhitelistInput() {
	app.activePopup = PopupNone
	app.pages.HidePage("whitelist-input")
	app.layout.SetFocus(0, app.tviewApp)
	app.requestsPanel.Refresh()
}

// openMapLocalPatternInput opens the map local pattern input
func (app *App) openMapLocalPatternInput() {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil {
		app.layout.SetStatus("[red]No request selected[-]")
		return
	}
	if flow.Response == nil {
		app.layout.SetStatus("[red]No response to map[-]")
		return
	}
	if flow.Tunneled {
		app.layout.SetStatus("[red]Cannot map tunneled connections[-]")
		return
	}

	app.activePopup = PopupMapLocalPattern
	app.mapLocalPatternInput.SetText(flow.Request.URL, true)
	app.pages.ShowPage("maplocal-pattern")
	app.tviewApp.SetFocus(app.mapLocalPatternInput)
}

// closeMapLocalPatternInput closes the map local pattern input
func (app *App) closeMapLocalPatternInput() {
	app.activePopup = PopupNone
	app.pages.HidePage("maplocal-pattern")
	app.layout.SetFocus(0, app.tviewApp)
}

// showWhitelistManager opens the whitelist manager modal
func (app *App) showWhitelistManager() {
	app.activePopup = PopupWhitelistManager
	app.whitelistManager.Refresh()
	app.pages.ShowPage("whitelist-manager")
	app.tviewApp.SetFocus(app.whitelistManager)
}

// addWhitelistFromManager opens an input to add a pattern from the whitelist manager
func (app *App) addWhitelistFromManager() {
	app.activePopup = PopupWhitelistInput
	app.pages.HidePage("whitelist-manager")

	app.whitelistInput.SetTitle(" Add Pattern ")
	app.whitelistInput.SetLabel(" Pattern (e.g., *.example.com): ")
	app.whitelistInput.SetText("")
	app.whitelistInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			pattern := app.whitelistInput.GetText()
			if pattern != "" {
				app.viewModel.AddWhitelistPattern(pattern)
				app.layout.SetStatus(fmt.Sprintf("[green]Added pattern: %s[-]", pattern))
			}
		}
		app.pages.HidePage("whitelist-input")
		// Return to whitelist manager
		app.showWhitelistManager()
	})

	app.pages.ShowPage("whitelist-input")
	app.tviewApp.SetFocus(app.whitelistInput)
}

// editWhitelistPattern opens an input to edit a whitelist pattern
func (app *App) editWhitelistPattern(oldPattern string) {
	app.activePopup = PopupWhitelistInput
	app.pages.HidePage("whitelist-manager")

	app.whitelistInput.SetBorder(true)
	app.whitelistInput.SetTitle(" Edit Pattern ")
	app.whitelistInput.SetTitleAlign(tview.AlignCenter)
	app.whitelistInput.SetLabel(" Pattern: ")
	app.whitelistInput.SetText(oldPattern)
	app.whitelistInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			newPattern := app.whitelistInput.GetText()
			if newPattern != "" && newPattern != oldPattern {
				app.viewModel.EditWhitelistPattern(oldPattern, newPattern)
				app.layout.SetStatus(fmt.Sprintf("[green]Updated pattern: %s → %s[-]", oldPattern, newPattern))
			}
		}
		// Restore original whitelist input style and behavior
		app.pages.HidePage("whitelist-input")
		app.whitelistInput.SetTitle(" Add Pattern ")
		app.whitelistInput.SetLabel(" Pattern (e.g., *.example.com): ")
		app.whitelistInput.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				pattern := app.whitelistInput.GetText()
				if pattern != "" {
					app.viewModel.AddWhitelistPattern(pattern)
					app.layout.SetStatus(fmt.Sprintf("[green]Added pattern: %s[-]", pattern))
				}
			}
			app.closeWhitelistInput()
		})
		// Return to whitelist manager
		app.showWhitelistManager()
	})

	app.pages.ShowPage("whitelist-input")
	app.tviewApp.SetFocus(app.whitelistInput)
}

// closeWhitelistManager closes the whitelist manager modal
func (app *App) closeWhitelistManager() {
	app.activePopup = PopupNone
	app.pages.HidePage("whitelist-manager")
	app.layout.SetFocus(0, app.tviewApp)
	app.requestsPanel.Refresh()
}

// showMapLocalManager opens the map local manager modal
func (app *App) showMapLocalManager() {
	app.activePopup = PopupMapLocalManager
	app.mapLocalManager.Refresh()
	app.pages.ShowPage("maplocal-manager")
	app.tviewApp.SetFocus(app.mapLocalManager)
}

// closeMapLocalManager closes the map local manager modal
func (app *App) closeMapLocalManager() {
	app.activePopup = PopupNone
	app.pages.HidePage("maplocal-manager")
	app.layout.SetFocus(0, app.tviewApp)
}

// openMapLocalForm opens the form to add a new map local rule
func (app *App) openMapLocalForm() {
	app.activePopup = PopupMapLocalForm
	app.pages.HidePage("maplocal-manager")
	app.pages.ShowPage("maplocal-form")
	app.tviewApp.SetFocus(app.mapLocalForm)
}

// closeMapLocalForm closes the map local form and returns to manager
func (app *App) closeMapLocalForm() {
	app.activePopup = PopupMapLocalManager
	app.pages.HidePage("maplocal-form")
	app.mapLocalManager.Refresh()
	app.pages.ShowPage("maplocal-manager")
	app.tviewApp.SetFocus(app.mapLocalManager)
}

// createMapLocalForm creates the form for adding map local rules
func (app *App) createMapLocalForm() {
	app.mapLocalForm = tview.NewForm()
	app.mapLocalForm.SetBorder(true)
	app.mapLocalForm.SetTitle(" Add Map Local Rule ")
	app.mapLocalForm.SetTitleAlign(tview.AlignCenter)
	app.mapLocalForm.SetFieldBackgroundColor(tcell.ColorDefault)
	app.mapLocalForm.SetFieldTextColor(tcell.ColorWhite)

	var pattern, localPath string

	app.mapLocalForm.AddTextArea("URL Pattern:", "", 60, 3, 0, func(text string) {
		pattern = text
	})
	app.mapLocalForm.AddInputField("Local Path:", "", 60, nil, func(text string) {
		localPath = text
	})
	app.mapLocalForm.AddButton("Add", func() {
		pattern = strings.TrimSpace(pattern)
		localPath = strings.TrimSpace(localPath)
		if pattern != "" && localPath != "" {
			app.viewModel.AddMapLocalRule(pattern, localPath, 200, "")
			app.layout.SetStatus(fmt.Sprintf("[green]Added mapping: %s → %s[-]", pattern, localPath))
		}
		app.closeMapLocalForm()
	})
	app.mapLocalForm.AddButton("Cancel", func() {
		app.closeMapLocalForm()
	})

	app.mapLocalForm.SetCancelFunc(func() {
		app.closeMapLocalForm()
	})
}

// showMapRemoteManager opens the map remote manager modal
func (app *App) showMapRemoteManager() {
	app.activePopup = PopupMapRemoteManager
	app.mapRemoteManager.Refresh()
	app.pages.ShowPage("mapremote-manager")
	app.tviewApp.SetFocus(app.mapRemoteManager)
}

// closeMapRemoteManager closes the map remote manager modal
func (app *App) closeMapRemoteManager() {
	app.activePopup = PopupNone
	app.pages.HidePage("mapremote-manager")
	app.layout.SetFocus(0, app.tviewApp)
}

// openMapRemoteForm opens the form to add a new map remote rule
func (app *App) openMapRemoteForm() {
	app.activePopup = PopupMapRemoteForm

	// Pre-fill URL pattern with selected flow's URL
	if flow := app.viewModel.GetSelectedFlow(); flow != nil && flow.Request != nil && !flow.Tunneled {
		if urlField, ok := app.mapRemoteForm.GetFormItem(0).(*tview.TextArea); ok {
			urlField.SetText(flow.Request.URL, true)
		}
	} else {
		if urlField, ok := app.mapRemoteForm.GetFormItem(0).(*tview.TextArea); ok {
			urlField.SetText("", false)
		}
	}
	// Clear the remote URL field
	if remoteField, ok := app.mapRemoteForm.GetFormItem(1).(*tview.InputField); ok {
		remoteField.SetText("")
	}

	app.pages.HidePage("mapremote-manager")
	app.pages.ShowPage("mapremote-form")
	app.tviewApp.SetFocus(app.mapRemoteForm)
}

// closeMapRemoteForm closes the map remote form and returns to manager
func (app *App) closeMapRemoteForm() {
	app.activePopup = PopupMapRemoteManager
	app.pages.HidePage("mapremote-form")
	app.mapRemoteManager.Refresh()
	app.pages.ShowPage("mapremote-manager")
	app.tviewApp.SetFocus(app.mapRemoteManager)
}

// editMapRemoteRule opens the form to edit an existing map remote rule
func (app *App) editMapRemoteRule(ruleID int) {
	rule := app.viewModel.GetMapRemoteRuleByID(ruleID)
	if rule == nil {
		return
	}

	// Clear and rebuild the form for editing
	app.mapRemoteForm.Clear(true)
	app.mapRemoteForm.SetTitle(" Edit Map Remote Rule ")

	var pattern, remoteURL string
	pattern = rule.Pattern
	remoteURL = rule.Replacement

	app.mapRemoteForm.AddTextArea("URL Pattern:", rule.Pattern, 60, 3, 0, func(text string) {
		pattern = text
	})
	app.mapRemoteForm.AddInputField("Remote URL:", rule.Replacement, 60, nil, func(text string) {
		remoteURL = text
	})
	app.mapRemoteForm.AddButton("Save", func() {
		pattern = strings.TrimSpace(pattern)
		remoteURL = strings.TrimSpace(remoteURL)
		if pattern != "" && remoteURL != "" {
			app.viewModel.UpdateMapRemoteRule(ruleID, pattern, remoteURL)
			app.layout.SetStatus(fmt.Sprintf("[green]Updated mapping: %s → %s[-]", pattern, remoteURL))
		}
		app.pages.HidePage("mapremote-form")
		app.mapRemoteManager.Refresh()
		app.pages.ShowPage("mapremote-manager")
		app.tviewApp.SetFocus(app.mapRemoteManager)
		// Reset form for next use (add mode)
		app.resetMapRemoteFormForAdd()
	})
	app.mapRemoteForm.AddButton("Cancel", func() {
		app.pages.HidePage("mapremote-form")
		app.pages.ShowPage("mapremote-manager")
		app.tviewApp.SetFocus(app.mapRemoteManager)
		// Reset form for next use (add mode)
		app.resetMapRemoteFormForAdd()
	})

	app.pages.HidePage("mapremote-manager")
	app.pages.ShowPage("mapremote-form")
	app.mapRemoteForm.SetFocus(0) // Focus on URL Pattern field
	app.tviewApp.SetFocus(app.mapRemoteForm)
}

// createMapRemoteForm creates the form for adding map remote rules (called once during init)
func (app *App) createMapRemoteForm() {
	app.mapRemoteForm = tview.NewForm()
	app.mapRemoteForm.SetBorder(true)
	app.mapRemoteForm.SetTitleAlign(tview.AlignCenter)
	app.mapRemoteForm.SetFieldBackgroundColor(tcell.ColorDefault)
	app.mapRemoteForm.SetFieldTextColor(tcell.ColorWhite)
	app.resetMapRemoteFormForAdd()
}

// resetMapRemoteFormForAdd clears and rebuilds the form for adding new rules
func (app *App) resetMapRemoteFormForAdd() {
	app.mapRemoteForm.Clear(true)
	app.mapRemoteForm.SetTitle(" Add Map Remote Rule ")

	var pattern, remoteURL string

	app.mapRemoteForm.AddTextArea("URL Pattern:", "", 60, 3, 0, func(text string) {
		pattern = text
	})
	app.mapRemoteForm.AddInputField("Remote URL:", "", 60, nil, func(text string) {
		remoteURL = text
	})
	app.mapRemoteForm.AddButton("Add", func() {
		pattern = strings.TrimSpace(pattern)
		remoteURL = strings.TrimSpace(remoteURL)
		if pattern != "" && remoteURL != "" {
			app.viewModel.AddMapRemoteRule(pattern, remoteURL)
			app.layout.SetStatus(fmt.Sprintf("[green]Added mapping: %s → %s[-]", pattern, remoteURL))
		}
		app.closeMapRemoteForm()
	})
	app.mapRemoteForm.AddButton("Cancel", func() {
		app.closeMapRemoteForm()
	})

	app.mapRemoteForm.SetCancelFunc(func() {
		app.closeMapRemoteForm()
	})
}

// quickMapRemote opens a prompt to map the selected request to a remote URL
func (app *App) quickMapRemote() {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil {
		app.layout.SetStatus("[red]No request selected[-]")
		return
	}

	if flow.Tunneled {
		app.layout.SetStatus("[red]Cannot map tunneled connections[-]")
		return
	}

	// Pre-fill the pattern with the selected URL
	pattern := flow.Request.URL

	// Create a quick input form
	app.mapRemoteForm.Clear(true)
	app.mapRemoteForm.SetTitle(" Map to Remote URL ")

	var remoteURL string

	app.mapRemoteForm.AddTextView("Pattern:", pattern, 50, 1, true, false)
	app.mapRemoteForm.AddInputField("Remote URL:", "", 50, nil, func(text string) {
		remoteURL = text
	})
	app.mapRemoteForm.AddButton("Add", func() {
		if remoteURL != "" {
			app.viewModel.AddMapRemoteRule(pattern, remoteURL)
			app.layout.SetStatus(fmt.Sprintf("[green]Mapped: %s → %s[-]", pattern, remoteURL))
		}
		app.activePopup = PopupNone
		app.pages.HidePage("mapremote-form")
		app.layout.SetFocus(0, app.tviewApp)
		// Reset form for next use
		app.resetMapRemoteFormForAdd()
	})
	app.mapRemoteForm.AddButton("Cancel", func() {
		app.activePopup = PopupNone
		app.pages.HidePage("mapremote-form")
		app.layout.SetFocus(0, app.tviewApp)
		// Reset form for next use
		app.resetMapRemoteFormForAdd()
	})

	app.activePopup = PopupMapRemoteForm
	app.pages.ShowPage("mapremote-form")
	app.tviewApp.SetFocus(app.mapRemoteForm)
}

// quickMapLocal creates a mapping from the selected request to a local file with the response
func (app *App) quickMapLocal() {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil {
		app.layout.SetStatus("[red]No request selected[-]")
		return
	}

	if flow.Response == nil {
		app.layout.SetStatus("[red]No response to map[-]")
		return
	}

	if flow.Tunneled {
		app.layout.SetStatus("[red]Cannot map tunneled connections[-]")
		return
	}

	pattern := flow.Request.URL
	localPath, err := app.writeJSONCMapping(flow, pattern, []string{
		fmt.Sprintf("  // Mapped from: %s", flow.Request.URL),
		fmt.Sprintf("  // Generated: %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]%s[-]", err))
		return
	}

	app.viewModel.AddMapLocalRule(pattern, localPath, flow.Response.StatusCode, flow.Response.Headers.Get("Content-Type"))
	app.layout.SetStatus(fmt.Sprintf("[green]Mapped to: %s[-]", localPath))
	app.openInEditor(localPath)
}

// createMapLocalWithPattern creates a mapping from the selected request to a local file using a custom pattern
func (app *App) createMapLocalWithPattern(pattern string) {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil || flow.Response == nil || flow.Tunneled {
		return
	}

	localPath, err := app.writeJSONCMapping(flow, pattern, []string{
		fmt.Sprintf("  // Mapped from: %s", flow.Request.URL),
		fmt.Sprintf("  // Pattern: %s", pattern),
		fmt.Sprintf("  // Generated: %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]%s[-]", err))
		return
	}

	app.viewModel.AddMapLocalRule(pattern, localPath, flow.Response.StatusCode, flow.Response.Headers.Get("Content-Type"))
	app.layout.SetStatus(fmt.Sprintf("[green]Mapped %s → %s[-]", pattern, localPath))
	app.openInEditor(localPath)
}

// writeJSONCMapping builds a JSONC response file from a flow's response,
// writes it to the mappings directory, and returns the file path.
func (app *App) writeJSONCMapping(flow *model.Flow, filenameSource string, comments []string) (string, error) {
	mappingsDir := filepath.Join(config.GetConfigDir(), "mappings")
	if err := os.MkdirAll(mappingsDir, 0o755); err != nil {
		return "", fmt.Errorf("Failed to create mappings dir: %s", err)
	}

	filename := generateFilename(filenameSource) + ".jsonc"
	localPath := filepath.Join(mappingsDir, filename)

	var buf bytes.Buffer
	buf.WriteString("{\n")
	for _, c := range comments {
		buf.WriteString(c)
		buf.WriteByte('\n')
	}
	buf.WriteString("\n")

	// Status
	statusText := strings.TrimPrefix(flow.Response.Status, fmt.Sprintf("%d ", flow.Response.StatusCode))
	buf.WriteString(fmt.Sprintf("  \"status\": %d,\n", flow.Response.StatusCode))
	buf.WriteString(fmt.Sprintf("  \"statusText\": %q,\n", statusText))
	buf.WriteString("\n")

	// Headers (skip Content-Length, will be recalculated)
	buf.WriteString("  \"headers\": {\n")
	headerCount := 0
	totalHeaders := len(flow.Response.Headers)
	for key, values := range flow.Response.Headers {
		headerCount++
		if strings.EqualFold(key, "Content-Length") {
			totalHeaders--
			continue
		}
		comma := ","
		if headerCount >= totalHeaders {
			comma = ""
		}
		if len(values) == 1 {
			buf.WriteString(fmt.Sprintf("    %q: %q%s\n", key, values[0], comma))
		} else {
			for i, v := range values {
				c := ","
				if headerCount >= totalHeaders && i == len(values)-1 {
					c = ""
				}
				buf.WriteString(fmt.Sprintf("    %q: %q%s\n", key, v, c))
			}
		}
	}
	buf.WriteString("  },\n\n")

	// Body
	buf.WriteString("  // Response body - edit below\n")
	body := flow.Response.Body
	contentType := flow.Response.Headers.Get("Content-Type")

	if strings.Contains(contentType, "json") || util.IsJSON(body) {
		var jsonObj interface{}
		if err := json.Unmarshal(body, &jsonObj); err == nil {
			if prettyBody, err := json.MarshalIndent(jsonObj, "  ", "  "); err == nil {
				buf.WriteString("  \"body\": ")
				buf.Write(prettyBody)
				buf.WriteString("\n")
			} else {
				buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
			}
		} else {
			buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
		}
	} else {
		buf.WriteString(fmt.Sprintf("  \"body\": %q\n", string(body)))
	}

	buf.WriteString("}\n")

	if err := os.WriteFile(localPath, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("Failed to write file: %s", err)
	}
	return localPath, nil
}

// generateFilename creates a safe filename from URL (without extension)
func generateFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("response_%d", time.Now().Unix())
	}

	// Start with host
	name := strings.ReplaceAll(parsed.Host, ":", "_")

	// Add path (sanitized)
	path := strings.Trim(parsed.Path, "/")
	if path != "" {
		path = strings.ReplaceAll(path, "/", "_")
		path = strings.ReplaceAll(path, "\\", "_")
		name += "_" + path
	}

	// Add query hash if present
	if parsed.RawQuery != "" {
		name += fmt.Sprintf("_%x", hash(parsed.RawQuery))
	}

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

// hash creates a simple hash of a string
func hash(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// getExtension returns file extension for content type
func getExtension(contentType string) string {
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "json"):
		return ".json"
	case strings.Contains(contentType, "javascript"):
		return ".js"
	case strings.Contains(contentType, "html"):
		return ".html"
	case strings.Contains(contentType, "css"):
		return ".css"
	case strings.Contains(contentType, "xml"):
		return ".xml"
	case strings.Contains(contentType, "text/plain"):
		return ".txt"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "image/svg"):
		return ".svg"
	default:
		return ".txt"
	}
}

// openInEditor opens a file in the user's preferred editor
func (app *App) openInEditor(filePath string) {
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi" // fallback
	}

	// Suspend the TUI and run the editor
	app.tviewApp.Suspend(func() {
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})
}

// showWhitelist displays current whitelist patterns
func (app *App) showWhitelist() {
	patterns := app.viewModel.GetWhitelistPatterns()
	if len(patterns) == 0 {
		app.layout.SetStatus("[yellow]Whitelist is empty. Press 'w' to add patterns.[-]")
		return
	}
	msg := "[cyan]Whitelist:[-] "
	for i, hp := range patterns {
		if i > 0 {
			msg += ", "
		}
		if hp.Enabled {
			msg += hp.Pattern
		} else {
			msg += "[gray]" + hp.Pattern + "[-]"
		}
	}
	app.layout.SetStatus(msg)
}

// ShowMessage shows a temporary message in the status bar
func (app *App) ShowMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	app.layout.SetStatus(msg)
}

// replaySelectedFlow replays the currently selected flow's request.
func (app *App) replaySelectedFlow() {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil {
		app.layout.SetStatus("[red]No request selected[-]")
		return
	}
	if flow.Tunneled {
		app.layout.SetStatus("[red]Cannot replay tunneled connections[-]")
		return
	}

	app.layout.SetStatus("[yellow]Replaying request...[-]")

	go func() {
		err := app.viewModel.ReplayFlow(flow)
		app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				app.layout.SetStatus(fmt.Sprintf("[red]Replay failed: %s[-]", err))
			} else {
				app.layout.SetStatus("[green]Request replayed[-]")
			}
		})
	}()
}

// copyCURL copies the selected flow as a cURL command to the clipboard.
func (app *App) copyCURL() {
	flow := app.viewModel.GetSelectedFlow()
	if flow == nil {
		app.layout.SetStatus("[red]No request selected[-]")
		return
	}

	curl, err := viewmodel.FormatCURL(flow)
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]%s[-]", err))
		return
	}

	if err := copyToClipboard(curl); err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]Clipboard error: %s[-]", err))
		return
	}
	app.layout.SetStatus("[green]cURL command copied to clipboard[-]")
}

// exportHAR exports flows as a HAR file. If all is true, exports all filtered
// flows; otherwise exports only the selected flow.
func (app *App) exportHAR(all bool) {
	var flows []*model.Flow
	var label string

	if all {
		flows = app.viewModel.GetFilteredFlows()
		label = fmt.Sprintf("%d flows", len(flows))
	} else {
		flow := app.viewModel.GetSelectedFlow()
		if flow == nil {
			app.layout.SetStatus("[red]No request selected[-]")
			return
		}
		flows = []*model.Flow{flow}
		label = "selected flow"
	}

	if len(flows) == 0 {
		app.layout.SetStatus("[red]No flows to export[-]")
		return
	}

	data, err := viewmodel.FormatHAR(flows)
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]HAR export failed: %s[-]", err))
		return
	}

	dir := os.TempDir()
	filename := fmt.Sprintf("proxy-tui-%s.har", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]Failed to write HAR: %s[-]", err))
		return
	}

	app.layout.SetStatus(fmt.Sprintf("[green]Exported %s to %s[-]", label, path))
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	// Try clipboard commands in priority order.
	// wl-copy first for Wayland (xsel/xclip may silently fail on Wayland).
	cmds := []struct {
		name string
		args []string
	}{
		{"wl-copy", nil},
		{"pbcopy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	}

	for _, c := range cmds {
		path, err := exec.LookPath(c.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard utility found (install xclip, xsel, or wl-copy)")
}

// showAlertManager opens the alert settings modal.
func (app *App) showAlertManager() {
	app.activePopup = PopupAlertManager
	app.alertManager.Refresh()
	app.pages.ShowPage("alert-manager")
	app.tviewApp.SetFocus(app.alertManager)
}

// closeAlertManager closes the alert settings modal.
func (app *App) closeAlertManager() {
	app.activePopup = PopupNone
	app.pages.HidePage("alert-manager")
	app.layout.SetFocus(0, app.tviewApp)
}

// openImportInput opens the HAR import file picker.
func (app *App) openImportInput() {
	app.activePopup = PopupImportHAR
	app.filePicker.Reset("")
	app.pages.ShowPage("import-har")
	app.tviewApp.SetFocus(app.filePicker)
}

// closeImportInput closes the HAR import input.
func (app *App) closeImportInput() {
	app.activePopup = PopupNone
	app.pages.HidePage("import-har")
	app.layout.SetFocus(0, app.tviewApp)
}

// doImportHAR reads and imports a HAR file.
func (app *App) doImportHAR(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]Failed to read file: %s[-]", err))
		return
	}

	flows, err := viewmodel.ParseHAR(data)
	if err != nil {
		app.layout.SetStatus(fmt.Sprintf("[red]%s[-]", err))
		return
	}

	if len(flows) == 0 {
		app.layout.SetStatus("[yellow]HAR file contains no entries[-]")
		return
	}

	count := app.viewModel.ImportFlows(flows)
	app.requestsPanel.Refresh()
	app.layout.SetStatus(fmt.Sprintf("[green]Imported %d flows from %s[-]", count, filepath.Base(path)))
}
