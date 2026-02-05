package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

const mapRemoteFile = "mapremote.jsonc"

// MapRemoteEntry represents a single map remote mapping
type MapRemoteEntry struct {
	Pattern   string `json:"pattern"`
	RemoteURL string `json:"remoteUrl"`
	Enabled   bool   `json:"enabled"`
}

// MapRemoteConfig represents the map remote configuration
type MapRemoteConfig struct {
	Mappings []MapRemoteEntry `json:"mappings"`
}

// GetMapRemotePath returns the path to the map remote file
func GetMapRemotePath() string {
	return filepath.Join(GetConfigDir(), mapRemoteFile)
}

// LoadMapRemote loads map remote entries from file
func LoadMapRemote() ([]MapRemoteEntry, error) {
	path := GetMapRemotePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []MapRemoteEntry{}, nil
		}
		return nil, err
	}

	// Strip comments from JSONC
	jsonData := stripJSONComments(data)

	var config MapRemoteConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return []MapRemoteEntry{}, nil
	}

	return config.Mappings, nil
}

// SaveMapRemote saves map remote entries to file
func SaveMapRemote(mappings []MapRemoteEntry) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := GetMapRemotePath()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write JSONC with comments
	file.WriteString("{\n")
	file.WriteString("  // Proxy TUI Map Remote\n")
	file.WriteString("  // Redirect requests to different URLs transparently\n")
	file.WriteString("  // Pattern supports wildcards: */api/users*, *example.com*\n")
	file.WriteString("  \"mappings\": [\n")

	for i, m := range mappings {
		comma := ","
		if i == len(mappings)-1 {
			comma = ""
		}

		entry := "    {"
		entry += "\"pattern\": \"" + escapeJSON(m.Pattern) + "\", "
		entry += "\"remoteUrl\": \"" + escapeJSON(m.RemoteURL) + "\", "
		entry += "\"enabled\": " + strconv.FormatBool(m.Enabled)
		entry += "}" + comma + "\n"
		file.WriteString(entry)
	}

	file.WriteString("  ]\n")
	file.WriteString("}\n")

	return nil
}

// AddMapRemoteEntry adds a mapping and saves
func AddMapRemoteEntry(entry MapRemoteEntry) error {
	mappings, err := LoadMapRemote()
	if err != nil {
		mappings = []MapRemoteEntry{}
	}

	// Check if pattern already exists
	for _, m := range mappings {
		if m.Pattern == entry.Pattern {
			return nil
		}
	}

	mappings = append(mappings, entry)
	return SaveMapRemote(mappings)
}

// RemoveMapRemoteEntry removes a mapping and saves
func RemoveMapRemoteEntry(pattern string) error {
	mappings, err := LoadMapRemote()
	if err != nil {
		return err
	}

	var newMappings []MapRemoteEntry
	for _, m := range mappings {
		if m.Pattern != pattern {
			newMappings = append(newMappings, m)
		}
	}

	return SaveMapRemote(newMappings)
}

// ToggleMapRemoteEntry toggles the enabled state of a mapping
func ToggleMapRemoteEntry(pattern string) error {
	mappings, err := LoadMapRemote()
	if err != nil {
		return err
	}

	for i, m := range mappings {
		if m.Pattern == pattern {
			mappings[i].Enabled = !m.Enabled
			break
		}
	}

	return SaveMapRemote(mappings)
}

// UpdateMapRemoteEntry updates a mapping by old pattern
func UpdateMapRemoteEntry(oldPattern string, entry MapRemoteEntry) error {
	mappings, err := LoadMapRemote()
	if err != nil {
		return err
	}

	for i, m := range mappings {
		if m.Pattern == oldPattern {
			mappings[i] = entry
			break
		}
	}

	return SaveMapRemote(mappings)
}
