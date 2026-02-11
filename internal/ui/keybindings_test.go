package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// --- Default bindings ---

func TestNewKeyBindings_NonEmpty(t *testing.T) {
	kb := NewKeyBindings()
	all := kb.GetAll()
	if len(all) == 0 {
		t.Error("default keybindings should be non-empty")
	}
}

// --- Context-specific lookup ---

func TestLookup_ContextSpecific(t *testing.T) {
	kb := NewKeyBindings()

	// 'j' in list context = next_item
	listBinding := kb.Lookup(tcell.KeyRune, 'j', ContextList)
	if listBinding == nil {
		t.Fatal("j in list should have a binding")
	}
	if listBinding.Action != ActionNextItem {
		t.Errorf("j in list = %q, want next_item", listBinding.Action)
	}

	// 'j' in detail context = scroll_down
	detailBinding := kb.Lookup(tcell.KeyRune, 'j', ContextDetail)
	if detailBinding == nil {
		t.Fatal("j in detail should have a binding")
	}
	if detailBinding.Action != ActionScrollDown {
		t.Errorf("j in detail = %q, want scroll_down", detailBinding.Action)
	}
}

// --- Global fallback ---

func TestLookup_GlobalFallback(t *testing.T) {
	kb := NewKeyBindings()

	// 'q' is global — should resolve from list context
	binding := kb.Lookup(tcell.KeyRune, 'q', ContextList)
	if binding == nil {
		t.Fatal("q should fall back to global")
	}
	if binding.Action != ActionQuit {
		t.Errorf("q = %q, want quit", binding.Action)
	}
}

// --- No match ---

func TestLookup_NoMatch(t *testing.T) {
	kb := NewKeyBindings()

	binding := kb.Lookup(tcell.KeyRune, 'Z', ContextList)
	if binding != nil {
		t.Error("unbound key should return nil")
	}
}

// --- Special keys ---

func TestLookup_SpecialKeys(t *testing.T) {
	kb := NewKeyBindings()

	// Tab
	tab := kb.Lookup(tcell.KeyTab, 0, ContextList)
	if tab == nil {
		t.Fatal("Tab should have a binding")
	}
	if tab.Action != ActionToggleFocus {
		t.Errorf("Tab = %q, want toggle_focus", tab.Action)
	}

	// Enter
	enter := kb.Lookup(tcell.KeyEnter, 0, ContextList)
	if enter == nil {
		t.Fatal("Enter should have a binding")
	}
	if enter.Action != ActionSelectItem {
		t.Errorf("Enter = %q, want select_item", enter.Action)
	}
}

// --- GetForContext ---

func TestGetForContext(t *testing.T) {
	kb := NewKeyBindings()

	listBindings := kb.GetForContext(ContextList)
	hasGlobal := false
	hasListSpecific := false
	for _, b := range listBindings {
		if b.Context == ContextGlobal {
			hasGlobal = true
		}
		if b.Context == ContextList {
			hasListSpecific = true
		}
	}
	if !hasGlobal {
		t.Error("GetForContext should include global bindings")
	}
	if !hasListSpecific {
		t.Error("GetForContext should include context-specific bindings")
	}
}

// --- FormatKey ---

func TestFormatKey(t *testing.T) {
	tests := []struct {
		name    string
		binding KeyBinding
		want    string
	}{
		{"rune", KeyBinding{Key: tcell.KeyRune, Rune: 'q'}, "q"},
		{"tab", KeyBinding{Key: tcell.KeyTab}, "Tab"},
		{"enter", KeyBinding{Key: tcell.KeyEnter}, "Enter"},
		{"up", KeyBinding{Key: tcell.KeyUp}, "\u2191"},
		{"down", KeyBinding{Key: tcell.KeyDown}, "\u2193"},
		{"pgup", KeyBinding{Key: tcell.KeyPgUp}, "PgUp"},
		{"pgdn", KeyBinding{Key: tcell.KeyPgDn}, "PgDn"},
		{"esc", KeyBinding{Key: tcell.KeyEsc}, "Esc"},
		{"unknown", KeyBinding{Key: tcell.KeyF1}, "?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatKey(tt.binding)
			if got != tt.want {
				t.Errorf("FormatKey = %q, want %q", got, tt.want)
			}
		})
	}
}
