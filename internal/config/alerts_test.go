package config

import (
	"testing"

	"nabu/internal/model"
)

func TestAlerts_RoundTrip(t *testing.T) {
	setupTestDir(t)

	rules := []model.AlertRule{
		{Type: model.AlertStatusCode, Value: 500, Enabled: true},
		{Type: model.AlertLatency, Value: 3000, Enabled: false},
	}

	if err := SaveAlerts(rules); err != nil {
		t.Fatalf("SaveAlerts: %v", err)
	}

	loaded, err := LoadAlerts()
	if err != nil {
		t.Fatalf("LoadAlerts: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(loaded))
	}
	if loaded[0].Type != model.AlertStatusCode || loaded[0].Value != 500 || !loaded[0].Enabled {
		t.Errorf("first rule: %+v", loaded[0])
	}
	if loaded[1].Type != model.AlertLatency || loaded[1].Value != 3000 || loaded[1].Enabled {
		t.Errorf("second rule: %+v", loaded[1])
	}
}

func TestLoadAlerts_NotFound(t *testing.T) {
	setupTestDir(t)

	rules, err := LoadAlerts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return defaults
	if len(rules) < 2 {
		t.Errorf("expected default rules, got %d", len(rules))
	}
}
