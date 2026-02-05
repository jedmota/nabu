package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

const mapLocalFile = "maplocal.jsonc"

// MapLocalEntry represents a single map local mapping
type MapLocalEntry struct {
	Pattern     string `json:"pattern"`
	LocalPath   string `json:"localPath"`
	Enabled     bool   `json:"enabled"`
	StatusCode  int    `json:"statusCode,omitempty"`  // optional, default 200
	ContentType string `json:"contentType,omitempty"` // optional, auto-detect if empty
}

// MapLocalConfig represents the map local configuration
type MapLocalConfig struct {
	Mappings []MapLocalEntry `json:"mappings"`
}

// GetMapLocalPath returns the path to the map local file
func GetMapLocalPath() string {
	return filepath.Join(GetConfigDir(), mapLocalFile)
}

// LoadMapLocal loads map local entries from file
func LoadMapLocal() ([]MapLocalEntry, error) {
	path := GetMapLocalPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []MapLocalEntry{}, nil
		}
		return nil, err
	}

	// Strip comments from JSONC
	jsonData := stripJSONComments(data)

	var config MapLocalConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return []MapLocalEntry{}, nil
	}

	return config.Mappings, nil
}

// SaveMapLocal saves map local entries to file
func SaveMapLocal(mappings []MapLocalEntry) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := GetMapLocalPath()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write JSONC with comments
	file.WriteString("{\n")
	file.WriteString("  // Proxy TUI Map Local\n")
	file.WriteString("  // Map remote URLs to local files\n")
	file.WriteString("  // Pattern supports wildcards: */api/users*, *example.com*\n")
	file.WriteString("  \"mappings\": [\n")

	for i, m := range mappings {
		comma := ","
		if i == len(mappings)-1 {
			comma = ""
		}

		statusCode := m.StatusCode
		if statusCode == 0 {
			statusCode = 200
		}

		entry := "    {"
		entry += "\"pattern\": \"" + escapeJSON(m.Pattern) + "\", "
		entry += "\"localPath\": \"" + escapeJSON(m.LocalPath) + "\", "
		entry += "\"enabled\": " + boolToString(m.Enabled)
		if m.StatusCode != 0 && m.StatusCode != 200 {
			entry += ", \"statusCode\": " + intToString(m.StatusCode)
		}
		if m.ContentType != "" {
			entry += ", \"contentType\": \"" + escapeJSON(m.ContentType) + "\""
		}
		entry += "}" + comma + "\n"
		file.WriteString(entry)
	}

	file.WriteString("  ]\n")
	file.WriteString("}\n")

	return nil
}

// intToString converts int to string
func intToString(i int) string {
	return strconv.Itoa(i)
}

// AddMapLocalEntry adds a mapping and saves
func AddMapLocalEntry(entry MapLocalEntry) error {
	mappings, err := LoadMapLocal()
	if err != nil {
		mappings = []MapLocalEntry{}
	}

	// Check if pattern already exists
	for _, m := range mappings {
		if m.Pattern == entry.Pattern {
			return nil
		}
	}

	if entry.StatusCode == 0 {
		entry.StatusCode = 200
	}

	mappings = append(mappings, entry)
	return SaveMapLocal(mappings)
}

// RemoveMapLocalEntry removes a mapping and saves
func RemoveMapLocalEntry(pattern string) error {
	mappings, err := LoadMapLocal()
	if err != nil {
		return err
	}

	var newMappings []MapLocalEntry
	for _, m := range mappings {
		if m.Pattern != pattern {
			newMappings = append(newMappings, m)
		}
	}

	return SaveMapLocal(newMappings)
}

// ToggleMapLocalEntry toggles the enabled state of a mapping
func ToggleMapLocalEntry(pattern string) error {
	mappings, err := LoadMapLocal()
	if err != nil {
		return err
	}

	for i, m := range mappings {
		if m.Pattern == pattern {
			mappings[i].Enabled = !m.Enabled
			break
		}
	}

	return SaveMapLocal(mappings)
}

// UpdateMapLocalEntry updates a mapping
func UpdateMapLocalEntry(pattern string, entry MapLocalEntry) error {
	mappings, err := LoadMapLocal()
	if err != nil {
		return err
	}

	for i, m := range mappings {
		if m.Pattern == pattern {
			mappings[i] = entry
			break
		}
	}

	return SaveMapLocal(mappings)
}
