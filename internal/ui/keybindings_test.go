package ui

import (
	"testing"
)

func TestDefaultKeyMap_HasBindings(t *testing.T) {
	km := DefaultKeyMap()

	// Verify core bindings have keys set
	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit should have key bindings")
	}
	if len(km.ToggleFocus.Keys()) == 0 {
		t.Error("ToggleFocus should have key bindings")
	}
	if len(km.Up.Keys()) == 0 {
		t.Error("Up should have key bindings")
	}
	if len(km.Down.Keys()) == 0 {
		t.Error("Down should have key bindings")
	}
}

func TestDefaultKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestDefaultKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()
	full := km.FullHelp()
	if len(full) == 0 {
		t.Error("FullHelp should return binding groups")
	}

	totalBindings := 0
	for _, group := range full {
		totalBindings += len(group)
	}
	if totalBindings < 10 {
		t.Errorf("FullHelp should have many bindings, got %d", totalBindings)
	}
}
