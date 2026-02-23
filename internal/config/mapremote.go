package config

import "path/filepath"

const mapRemoteFile = "mapremote.jsonc"

// MapRemoteEntry represents a single map remote mapping
type MapRemoteEntry struct {
	Pattern   string `json:"pattern"`
	RemoteURL string `json:"remoteUrl"`
	Enabled   bool   `json:"enabled"`
	Method    string `json:"method,omitempty"` // optional, empty = match all
}

func (e MapRemoteEntry) GetPattern() string { return e.Pattern }
func (e MapRemoteEntry) GetEnabled() bool   { return e.Enabled }

var mapRemoteStore = ConfigStore[MapRemoteEntry]{
	filename: mapRemoteFile,
	header: []string{
		"// Proxy TUI Map Remote",
		"// Redirect requests to different URLs transparently",
		"// Pattern supports wildcards: */api/users*, *example.com*",
	},
	wrapKey: "mappings",
	toggleFn: func(e MapRemoteEntry) MapRemoteEntry {
		e.Enabled = !e.Enabled
		return e
	},
}

// GetMapRemotePath returns the path to the map remote file
func GetMapRemotePath() string {
	return filepath.Join(GetConfigDir(), mapRemoteFile)
}

// LoadMapRemote loads map remote entries from file
func LoadMapRemote() ([]MapRemoteEntry, error) { return mapRemoteStore.Load() }

// SaveMapRemote saves map remote entries to file
func SaveMapRemote(mappings []MapRemoteEntry) error { return mapRemoteStore.Save(mappings) }

// AddMapRemoteEntry adds a mapping and saves
func AddMapRemoteEntry(entry MapRemoteEntry) error { return mapRemoteStore.Add(entry) }

// RemoveMapRemoteEntry removes a mapping and saves
func RemoveMapRemoteEntry(pattern string) error { return mapRemoteStore.Remove(pattern) }

// ToggleMapRemoteEntry toggles the enabled state of a mapping
func ToggleMapRemoteEntry(pattern string) error { return mapRemoteStore.Toggle(pattern) }

// UpdateMapRemoteEntry updates a mapping by old pattern
func UpdateMapRemoteEntry(oldPattern string, entry MapRemoteEntry) error {
	return mapRemoteStore.Update(oldPattern, entry)
}
