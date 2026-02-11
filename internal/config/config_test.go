package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	configDirOverride = dir
	t.Cleanup(func() { configDirOverride = "" })
}

// --- stripJSONComments ---

func TestStripJSONComments_LineComment(t *testing.T) {
	input := "// comment\n{\"key\": \"value\"}\n"
	got := string(stripJSONComments([]byte(input)))
	if strings.Contains(got, "// comment") {
		t.Error("line comment should be stripped")
	}
	if !strings.Contains(got, `"key": "value"`) {
		t.Error("JSON content should be preserved")
	}
}

func TestStripJSONComments_InsideString(t *testing.T) {
	input := `{"url": "http://example.com//path"}` + "\n"
	got := string(stripJSONComments([]byte(input)))
	if !strings.Contains(got, "http://example.com//path") {
		t.Error("// inside JSON string should be preserved")
	}
}

// --- SaveWhitelist / LoadWhitelist round-trip ---

func TestWhitelist_RoundTrip(t *testing.T) {
	setupTestDir(t)

	patterns := []WhitelistPattern{
		{Pattern: "*.example.com", Enabled: true},
		{Pattern: "other.com", Enabled: false},
	}
	if err := SaveWhitelist(patterns); err != nil {
		t.Fatalf("SaveWhitelist: %v", err)
	}

	loaded, err := LoadWhitelist()
	if err != nil {
		t.Fatalf("LoadWhitelist: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d patterns, want 2", len(loaded))
	}
	if loaded[0].Pattern != "*.example.com" || !loaded[0].Enabled {
		t.Errorf("first pattern: %+v", loaded[0])
	}
	if loaded[1].Pattern != "other.com" || loaded[1].Enabled {
		t.Errorf("second pattern: %+v", loaded[1])
	}
}

// --- LoadWhitelist file not found ---

func TestLoadWhitelist_NotFound(t *testing.T) {
	setupTestDir(t)

	patterns, err := LoadWhitelist()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected empty patterns, got %d", len(patterns))
	}
}

// --- LoadWhitelist migrate old JSON format ---

func TestLoadWhitelist_MigrateOldJSON(t *testing.T) {
	setupTestDir(t)

	content := `{"patterns": ["a.com", "b.com"]}`
	path := GetWhitelistPath()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := LoadWhitelist()
	if err != nil {
		t.Fatalf("LoadWhitelist: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
	if !patterns[0].Enabled {
		t.Error("migrated patterns should be enabled")
	}
}

// --- LoadWhitelist migrate plain text ---

func TestLoadWhitelist_MigratePlainText(t *testing.T) {
	setupTestDir(t)

	content := "a.com\n# comment\nb.com\n"
	path := GetWhitelistPath()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := LoadWhitelist()
	if err != nil {
		t.Fatalf("LoadWhitelist: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
}

// --- AddToWhitelist ---

func TestAddToWhitelist(t *testing.T) {
	setupTestDir(t)

	if err := AddToWhitelist("a.com"); err != nil {
		t.Fatal(err)
	}
	// Duplicate should be no-op
	if err := AddToWhitelist("a.com"); err != nil {
		t.Fatal(err)
	}

	patterns, _ := LoadWhitelist()
	if len(patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(patterns))
	}
}

// --- RemoveFromWhitelist ---

func TestRemoveFromWhitelist(t *testing.T) {
	setupTestDir(t)

	AddToWhitelist("a.com")
	AddToWhitelist("b.com")
	RemoveFromWhitelist("a.com")

	patterns, _ := LoadWhitelist()
	if len(patterns) != 1 || patterns[0].Pattern != "b.com" {
		t.Errorf("after Remove, patterns = %+v", patterns)
	}
}

// --- EditWhitelistPattern ---

func TestEditWhitelistPattern(t *testing.T) {
	setupTestDir(t)

	AddToWhitelist("old.com")
	EditWhitelistPattern("old.com", "new.com")

	patterns, _ := LoadWhitelist()
	if len(patterns) != 1 || patterns[0].Pattern != "new.com" {
		t.Errorf("after Edit, patterns = %+v", patterns)
	}
}

// --- ToggleWhitelistPattern ---

func TestToggleWhitelistPattern(t *testing.T) {
	setupTestDir(t)

	AddToWhitelist("a.com")
	ToggleWhitelistPattern("a.com")

	patterns, _ := LoadWhitelist()
	if len(patterns) != 1 || patterns[0].Enabled {
		t.Errorf("after Toggle, pattern should be disabled: %+v", patterns)
	}
}

// --- SetWhitelistPatternEnabled ---

func TestSetWhitelistPatternEnabled(t *testing.T) {
	setupTestDir(t)

	AddToWhitelist("a.com")
	SetWhitelistPatternEnabled("a.com", false)

	patterns, _ := LoadWhitelist()
	if patterns[0].Enabled {
		t.Error("should be disabled")
	}

	SetWhitelistPatternEnabled("a.com", true)
	patterns, _ = LoadWhitelist()
	if !patterns[0].Enabled {
		t.Error("should be enabled")
	}
}

// --- ClearWhitelist ---

func TestClearWhitelist(t *testing.T) {
	setupTestDir(t)

	AddToWhitelist("a.com")
	AddToWhitelist("b.com")
	ClearWhitelist()

	patterns, _ := LoadWhitelist()
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns after Clear, got %d", len(patterns))
	}
}

// --- MapLocal round-trip ---

func TestMapLocal_RoundTrip(t *testing.T) {
	setupTestDir(t)

	entries := []MapLocalEntry{
		{Pattern: "*/api/*", LocalPath: "/tmp/mock.json", Enabled: true, StatusCode: 200},
	}
	if err := SaveMapLocal(entries); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadMapLocal()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Pattern != "*/api/*" {
		t.Errorf("loaded = %+v", loaded)
	}
}

func TestLoadMapLocal_NotFound(t *testing.T) {
	setupTestDir(t)

	entries, err := LoadMapLocal()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

func TestMapLocal_AddRemoveToggleUpdate(t *testing.T) {
	setupTestDir(t)

	AddMapLocalEntry(MapLocalEntry{Pattern: "p1", LocalPath: "/tmp/a", Enabled: true})
	AddMapLocalEntry(MapLocalEntry{Pattern: "p2", LocalPath: "/tmp/b", Enabled: true})

	// Duplicate add is no-op
	AddMapLocalEntry(MapLocalEntry{Pattern: "p1", LocalPath: "/tmp/c", Enabled: true})
	entries, _ := LoadMapLocal()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Toggle
	ToggleMapLocalEntry("p1")
	entries, _ = LoadMapLocal()
	for _, e := range entries {
		if e.Pattern == "p1" && e.Enabled {
			t.Error("p1 should be disabled after toggle")
		}
	}

	// Update
	UpdateMapLocalEntry("p1", MapLocalEntry{Pattern: "p1-updated", LocalPath: "/tmp/updated", Enabled: true})
	entries, _ = LoadMapLocal()
	found := false
	for _, e := range entries {
		if e.Pattern == "p1-updated" {
			found = true
		}
	}
	if !found {
		t.Error("update should have replaced p1 with p1-updated")
	}

	// Remove
	RemoveMapLocalEntry("p2")
	entries, _ = LoadMapLocal()
	if len(entries) != 1 {
		t.Errorf("after Remove, expected 1 entry, got %d", len(entries))
	}
}

// --- MapRemote round-trip ---

func TestMapRemote_RoundTrip(t *testing.T) {
	setupTestDir(t)

	entries := []MapRemoteEntry{
		{Pattern: "*/api/*", RemoteURL: "http://localhost:3000", Enabled: true},
	}
	if err := SaveMapRemote(entries); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadMapRemote()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Pattern != "*/api/*" {
		t.Errorf("loaded = %+v", loaded)
	}
}

func TestLoadMapRemote_NotFound(t *testing.T) {
	setupTestDir(t)

	entries, err := LoadMapRemote()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %d", len(entries))
	}
}

func TestMapRemote_AddRemoveToggleUpdate(t *testing.T) {
	setupTestDir(t)

	AddMapRemoteEntry(MapRemoteEntry{Pattern: "p1", RemoteURL: "http://a.com", Enabled: true})
	AddMapRemoteEntry(MapRemoteEntry{Pattern: "p2", RemoteURL: "http://b.com", Enabled: true})

	// Duplicate
	AddMapRemoteEntry(MapRemoteEntry{Pattern: "p1", RemoteURL: "http://c.com", Enabled: true})
	entries, _ := LoadMapRemote()
	if len(entries) != 2 {
		t.Errorf("expected 2, got %d", len(entries))
	}

	// Toggle
	ToggleMapRemoteEntry("p1")
	entries, _ = LoadMapRemote()
	for _, e := range entries {
		if e.Pattern == "p1" && e.Enabled {
			t.Error("p1 should be disabled")
		}
	}

	// Update
	UpdateMapRemoteEntry("p1", MapRemoteEntry{Pattern: "p1-new", RemoteURL: "http://new.com", Enabled: true})
	entries, _ = LoadMapRemote()
	found := false
	for _, e := range entries {
		if e.Pattern == "p1-new" {
			found = true
		}
	}
	if !found {
		t.Error("update should replace p1")
	}

	// Remove
	RemoveMapRemoteEntry("p2")
	entries, _ = LoadMapRemote()
	if len(entries) != 1 {
		t.Errorf("after Remove, expected 1, got %d", len(entries))
	}
}

// --- escapeJSON ---

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`simple`, `simple`},
		{`back\slash`, `back\\slash`},
		{`has"quote`, `has\"quote`},
		{`both\"`, `both\\\"`},
	}
	for _, tt := range tests {
		got := escapeJSON(tt.in)
		if got != tt.want {
			t.Errorf("escapeJSON(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- JSONC round-trip with // in pattern value ---

func TestWhitelist_JSONC_SlashInPattern(t *testing.T) {
	setupTestDir(t)

	patterns := []WhitelistPattern{
		{Pattern: "http://example.com//api", Enabled: true},
	}
	if err := SaveWhitelist(patterns); err != nil {
		t.Fatal(err)
	}

	// Verify the file contains the pattern with //
	data, _ := os.ReadFile(filepath.Join(GetConfigDir(), whitelistFile))
	if !strings.Contains(string(data), "http://example.com//api") {
		t.Error("saved file should contain the literal // in the pattern value")
	}

	loaded, err := LoadWhitelist()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Pattern != "http://example.com//api" {
		t.Errorf("round-trip failed: %+v", loaded)
	}
}
