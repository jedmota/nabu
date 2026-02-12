package config

import "path/filepath"

const mapLocalFile = "maplocal.jsonc"

// MapLocalEntry represents a single map local mapping
type MapLocalEntry struct {
	Pattern     string `json:"pattern"`
	LocalPath   string `json:"localPath"`
	Enabled     bool   `json:"enabled"`
	StatusCode  int    `json:"statusCode,omitempty"`  // optional, default 200
	ContentType string `json:"contentType,omitempty"` // optional, auto-detect if empty
}

func (e MapLocalEntry) GetPattern() string { return e.Pattern }
func (e MapLocalEntry) GetEnabled() bool   { return e.Enabled }

var mapLocalStore = ConfigStore[MapLocalEntry]{
	filename: mapLocalFile,
	header: []string{
		"// Proxy TUI Map Local",
		"// Map remote URLs to local files",
		"// Pattern supports wildcards: */api/users*, *example.com*",
	},
	wrapKey: "mappings",
	toggleFn: func(e MapLocalEntry) MapLocalEntry {
		e.Enabled = !e.Enabled
		return e
	},
}

// GetMapLocalPath returns the path to the map local file
func GetMapLocalPath() string {
	return filepath.Join(GetConfigDir(), mapLocalFile)
}

// LoadMapLocal loads map local entries from file
func LoadMapLocal() ([]MapLocalEntry, error) { return mapLocalStore.Load() }

// SaveMapLocal saves map local entries to file
func SaveMapLocal(mappings []MapLocalEntry) error { return mapLocalStore.Save(mappings) }

// AddMapLocalEntry adds a mapping and saves
func AddMapLocalEntry(entry MapLocalEntry) error {
	if entry.StatusCode == 0 {
		entry.StatusCode = 200
	}
	return mapLocalStore.Add(entry)
}

// RemoveMapLocalEntry removes a mapping and saves
func RemoveMapLocalEntry(pattern string) error { return mapLocalStore.Remove(pattern) }

// ToggleMapLocalEntry toggles the enabled state of a mapping
func ToggleMapLocalEntry(pattern string) error { return mapLocalStore.Toggle(pattern) }

// UpdateMapLocalEntry updates a mapping
func UpdateMapLocalEntry(pattern string, entry MapLocalEntry) error {
	return mapLocalStore.Update(pattern, entry)
}
