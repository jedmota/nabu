package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Global
	Quit        key.Binding
	ToggleFocus key.Binding
	ExpandPanel key.Binding
	ToggleHelp  key.Binding

	// Navigation
	Up       key.Binding
	Down     key.Binding
	GoToTop  key.Binding // gg chord handled separately
	GoToBot  key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Filters
	FilterAll      key.Binding
	FilterWhite    key.Binding
	FilterStarred  key.Binding
	Search         key.Binding

	// Whitelist
	AddWhitelist   key.Binding
	ShowWhitelist  key.Binding
	ClearWhitelist key.Binding

	// Mapping
	QuickMapLocal   key.Binding
	MapLocalManager key.Binding
	AddMapRemote    key.Binding
	MapRemoteManager key.Binding

	// Flow operations
	Replay       key.Binding
	CopyCURL     key.Binding
	ExportHAR    key.Binding
	ExportAllHAR key.Binding
	ImportHAR    key.Binding
	Pause        key.Binding
	Star         key.Binding
	StarAll      key.Binding
	ClearFlows   key.Binding

	// Alerts
	Alerts key.Binding

	// Detail
	ToggleRaw key.Binding

	// Theme
	CycleTheme key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:        key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
		ToggleFocus: key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "Toggle focus")),
		ExpandPanel: key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "Expand panel")),
		ToggleHelp:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "Toggle help")),

		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/up", "Up")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/down", "Down")),
		GoToBot:  key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "Go to bottom")),
		PageUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("PgUp", "Page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("PgDn", "Page down")),

		FilterAll:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "Filter: All")),
		FilterWhite:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Filter: Whitelist")),
		FilterStarred: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "Filter: Starred")),
		Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "Filter: Custom")),

		AddWhitelist:   key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "Add to whitelist")),
		ShowWhitelist:  key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "Show whitelist")),
		ClearWhitelist: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "Clear whitelist")),

		QuickMapLocal:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "Map selected to local")),
		MapLocalManager:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "Map local manager")),
		AddMapRemote:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Add map remote rule")),
		MapRemoteManager: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "Map remote manager")),

		Replay:       key.NewBinding(key.WithKeys("."), key.WithHelp(".", "Replay request")),
		CopyCURL:     key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "Copy as cURL")),
		ExportHAR:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "Export selected as HAR")),
		ExportAllHAR: key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "Export all as HAR")),
		ImportHAR:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "Import HAR file")),
		Pause:        key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "Pause/resume")),
		Star:         key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "Star selected")),
		StarAll:      key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "Star all listed")),
		ClearFlows:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "Clear flows")),

		Alerts: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "Alert settings")),

		ToggleRaw:  key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "Toggle raw/pretty")),
		CycleTheme: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "Cycle theme")),
	}
}

// ShortHelp returns keybindings to show in the mini help.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.ToggleHelp, k.Quit}
}

// FullHelp returns keybindings grouped for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.ToggleFocus, k.ExpandPanel, k.ToggleHelp},
		{k.Up, k.Down, k.PageUp, k.PageDown, k.GoToBot},
		{k.FilterAll, k.FilterWhite, k.FilterStarred, k.Search},
		{k.AddWhitelist, k.ShowWhitelist, k.ClearWhitelist},
		{k.QuickMapLocal, k.MapLocalManager, k.AddMapRemote, k.MapRemoteManager},
		{k.Replay, k.CopyCURL, k.ExportHAR, k.ExportAllHAR, k.ImportHAR},
		{k.Pause, k.Star, k.StarAll, k.ClearFlows},
		{k.Alerts, k.ToggleRaw, k.CycleTheme},
	}
}
