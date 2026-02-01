package ui

import "github.com/gdamore/tcell/v2"

// Context represents the current UI context for keybindings
type Context string

const (
	ContextGlobal Context = "global"
	ContextList   Context = "list"
	ContextDetail Context = "detail"
)

// Action represents a keybinding action
type Action string

const (
	// Global actions
	ActionQuit        Action = "quit"
	ActionToggleFocus Action = "toggle_focus"
	ActionExpandPanel Action = "expand_panel"
	ActionToggleHelp  Action = "toggle_help"

	// List actions
	ActionNextItem     Action = "next_item"
	ActionPrevItem     Action = "prev_item"
	ActionSelectItem   Action = "select_item"
	ActionRefresh      Action = "refresh"
	ActionSearch       Action = "search"
	ActionFilterAll    Action = "filter_all"
	ActionFilterWhite  Action = "filter_white"
	ActionClearFlows   Action = "clear_flows"

	// Detail actions
	ActionToggleRaw    Action = "toggle_raw"
	ActionScrollUp     Action = "scroll_up"
	ActionScrollDown   Action = "scroll_down"
	ActionPageUp       Action = "page_up"
	ActionPageDown     Action = "page_down"
)

// KeyBinding represents a single keybinding
type KeyBinding struct {
	Key         tcell.Key
	Rune        rune
	Context     Context
	Action      Action
	Description string
}

// KeyBindings holds all keybindings
type KeyBindings struct {
	bindings []KeyBinding
	byKey    map[string][]KeyBinding
}

// NewKeyBindings creates the default keybindings
func NewKeyBindings() *KeyBindings {
	kb := &KeyBindings{
		bindings: make([]KeyBinding, 0),
		byKey:    make(map[string][]KeyBinding),
	}

	// Global bindings
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'q', Context: ContextGlobal, Action: ActionQuit, Description: "Quit"})
	kb.Add(KeyBinding{Key: tcell.KeyTab, Context: ContextGlobal, Action: ActionToggleFocus, Description: "Toggle focus"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'H', Context: ContextGlobal, Action: ActionExpandPanel, Description: "Expand panel"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: '?', Context: ContextGlobal, Action: ActionToggleHelp, Description: "Toggle help"})

	// List bindings
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'j', Context: ContextList, Action: ActionNextItem, Description: "Next item"})
	kb.Add(KeyBinding{Key: tcell.KeyDown, Context: ContextList, Action: ActionNextItem, Description: "Next item"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'k', Context: ContextList, Action: ActionPrevItem, Description: "Previous item"})
	kb.Add(KeyBinding{Key: tcell.KeyUp, Context: ContextList, Action: ActionPrevItem, Description: "Previous item"})
	kb.Add(KeyBinding{Key: tcell.KeyEnter, Context: ContextList, Action: ActionSelectItem, Description: "View detail"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'r', Context: ContextList, Action: ActionRefresh, Description: "Refresh"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: '/', Context: ContextList, Action: ActionSearch, Description: "Search"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: '1', Context: ContextList, Action: ActionFilterAll, Description: "Filter: All"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: '2', Context: ContextList, Action: ActionFilterWhite, Description: "Filter: Whitelist"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'c', Context: ContextList, Action: ActionClearFlows, Description: "Clear flows"})

	// Detail bindings
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'T', Context: ContextDetail, Action: ActionToggleRaw, Description: "Toggle raw/pretty"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'j', Context: ContextDetail, Action: ActionScrollDown, Description: "Scroll down"})
	kb.Add(KeyBinding{Key: tcell.KeyDown, Context: ContextDetail, Action: ActionScrollDown, Description: "Scroll down"})
	kb.Add(KeyBinding{Key: tcell.KeyRune, Rune: 'k', Context: ContextDetail, Action: ActionScrollUp, Description: "Scroll up"})
	kb.Add(KeyBinding{Key: tcell.KeyUp, Context: ContextDetail, Action: ActionScrollUp, Description: "Scroll up"})
	kb.Add(KeyBinding{Key: tcell.KeyPgDn, Context: ContextDetail, Action: ActionPageDown, Description: "Page down"})
	kb.Add(KeyBinding{Key: tcell.KeyPgUp, Context: ContextDetail, Action: ActionPageUp, Description: "Page up"})

	return kb
}

// Add adds a keybinding
func (kb *KeyBindings) Add(binding KeyBinding) {
	kb.bindings = append(kb.bindings, binding)

	key := kb.makeKey(binding.Key, binding.Rune)
	kb.byKey[key] = append(kb.byKey[key], binding)
}

// makeKey creates a unique key for lookup
func (kb *KeyBindings) makeKey(key tcell.Key, r rune) string {
	if key != tcell.KeyRune {
		return string(key)
	}
	return string(r)
}

// Lookup finds a matching keybinding
func (kb *KeyBindings) Lookup(key tcell.Key, r rune, context Context) *KeyBinding {
	k := kb.makeKey(key, r)
	bindings, ok := kb.byKey[k]
	if !ok {
		return nil
	}

	// Check context-specific first
	for i := range bindings {
		if bindings[i].Context == context {
			return &bindings[i]
		}
	}

	// Check global
	for i := range bindings {
		if bindings[i].Context == ContextGlobal {
			return &bindings[i]
		}
	}

	return nil
}

// GetForContext returns all bindings for a context
func (kb *KeyBindings) GetForContext(context Context) []KeyBinding {
	result := make([]KeyBinding, 0)
	for _, b := range kb.bindings {
		if b.Context == context || b.Context == ContextGlobal {
			result = append(result, b)
		}
	}
	return result
}

// GetAll returns all bindings
func (kb *KeyBindings) GetAll() []KeyBinding {
	return kb.bindings
}

// FormatKey returns a human-readable key name
func FormatKey(binding KeyBinding) string {
	if binding.Key == tcell.KeyRune {
		return string(binding.Rune)
	}

	switch binding.Key {
	case tcell.KeyTab:
		return "Tab"
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyUp:
		return "↑"
	case tcell.KeyDown:
		return "↓"
	case tcell.KeyLeft:
		return "←"
	case tcell.KeyRight:
		return "→"
	case tcell.KeyPgUp:
		return "PgUp"
	case tcell.KeyPgDn:
		return "PgDn"
	case tcell.KeyEsc:
		return "Esc"
	default:
		return "?"
	}
}
