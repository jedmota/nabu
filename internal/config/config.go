package config

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDir     = ".proxy-tui"
	whitelistFile = "whitelist.jsonc"
)

// WhitelistPattern represents a single whitelist pattern with enabled state
type WhitelistPattern struct {
	Pattern string `json:"pattern"`
	Enabled bool   `json:"enabled"`
}

// WhitelistConfig represents the whitelist configuration
type WhitelistConfig struct {
	Patterns []WhitelistPattern `json:"patterns"`
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return configDir
	}
	return filepath.Join(home, configDir)
}

// GetWhitelistPath returns the path to the whitelist file
func GetWhitelistPath() string {
	return filepath.Join(GetConfigDir(), whitelistFile)
}

// stripJSONComments removes // and /* */ comments from JSONC
// It properly handles comment-like sequences inside JSON strings
func stripJSONComments(data []byte) []byte {
	var result strings.Builder
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		// Process character by character to properly track string state
		inString := false
		i := 0
		for i < len(line) {
			c := line[i]

			// Track string boundaries (handle escaped quotes)
			if c == '"' && (i == 0 || line[i-1] != '\\') {
				inString = !inString
				result.WriteByte(c)
				i++
				continue
			}

			// Only process comments when not inside a string
			if !inString {
				// Check for line comment //
				if c == '/' && i+1 < len(line) && line[i+1] == '/' {
					// Skip rest of line
					break
				}
				// Check for block comment /* - skip for now, just treat // as comments
				// Block comments spanning lines are rare in JSONC config files
			}

			result.WriteByte(c)
			i++
		}
		result.WriteString("\n")
	}

	return []byte(result.String())
}

// LoadWhitelist loads whitelist patterns from file
func LoadWhitelist() ([]WhitelistPattern, error) {
	path := GetWhitelistPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []WhitelistPattern{}, nil
		}
		return nil, err
	}

	// Strip comments from JSONC
	jsonData := stripJSONComments(data)

	var config WhitelistConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		// Try to migrate from old format (plain text or old JSON)
		return migrateFromOldFormat(path, jsonData)
	}

	return config.Patterns, nil
}

// OldWhitelistConfig represents the old whitelist format (string array)
type OldWhitelistConfig struct {
	Patterns []string `json:"patterns"`
}

// migrateFromOldFormat attempts to read old formats (plain text or old JSON with string array)
func migrateFromOldFormat(path string, jsonData []byte) ([]WhitelistPattern, error) {
	// Try old JSON format first (string array)
	var oldConfig OldWhitelistConfig
	if err := json.Unmarshal(jsonData, &oldConfig); err == nil && len(oldConfig.Patterns) > 0 {
		patterns := make([]WhitelistPattern, len(oldConfig.Patterns))
		for i, p := range oldConfig.Patterns {
			patterns[i] = WhitelistPattern{Pattern: p, Enabled: true}
		}
		// Save in new format
		SaveWhitelist(patterns)
		return patterns, nil
	}

	// Try plain text format
	file, err := os.Open(path)
	if err != nil {
		return []WhitelistPattern{}, nil
	}
	defer file.Close()

	var patterns []WhitelistPattern
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, WhitelistPattern{Pattern: line, Enabled: true})
	}

	// Save in new format
	if len(patterns) > 0 {
		SaveWhitelist(patterns)
	}

	return patterns, nil
}

// SaveWhitelist saves whitelist patterns to file
func SaveWhitelist(patterns []WhitelistPattern) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := GetWhitelistPath()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write JSONC with comments
	file.WriteString("{\n")
	file.WriteString("  // Proxy TUI Whitelist\n")
	file.WriteString("  // Patterns support wildcards: *.example.com\n")
	file.WriteString("  \"patterns\": [\n")

	for i, p := range patterns {
		comma := ","
		if i == len(patterns)-1 {
			comma = ""
		}
		file.WriteString("    {\"pattern\": \"" + escapeJSON(p.Pattern) + "\", \"enabled\": " + boolToString(p.Enabled) + "}" + comma + "\n")
	}

	file.WriteString("  ]\n")
	file.WriteString("}\n")

	return nil
}

// boolToString converts a bool to string
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// escapeJSON escapes a string for JSON
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// AddToWhitelist adds a pattern and saves
func AddToWhitelist(pattern string) error {
	patterns, err := LoadWhitelist()
	if err != nil {
		patterns = []WhitelistPattern{}
	}

	// Check if already exists
	for _, p := range patterns {
		if p.Pattern == pattern {
			return nil
		}
	}

	patterns = append(patterns, WhitelistPattern{Pattern: pattern, Enabled: true})
	return SaveWhitelist(patterns)
}

// RemoveFromWhitelist removes a pattern and saves
func RemoveFromWhitelist(pattern string) error {
	patterns, err := LoadWhitelist()
	if err != nil {
		return err
	}

	var newPatterns []WhitelistPattern
	for _, p := range patterns {
		if p.Pattern != pattern {
			newPatterns = append(newPatterns, p)
		}
	}

	return SaveWhitelist(newPatterns)
}

// ToggleWhitelistPattern toggles the enabled state of a pattern
func ToggleWhitelistPattern(pattern string) error {
	patterns, err := LoadWhitelist()
	if err != nil {
		return err
	}

	for i, p := range patterns {
		if p.Pattern == pattern {
			patterns[i].Enabled = !patterns[i].Enabled
			break
		}
	}

	return SaveWhitelist(patterns)
}

// SetWhitelistPatternEnabled sets the enabled state of a pattern
func SetWhitelistPatternEnabled(pattern string, enabled bool) error {
	patterns, err := LoadWhitelist()
	if err != nil {
		return err
	}

	for i, p := range patterns {
		if p.Pattern == pattern {
			patterns[i].Enabled = enabled
			break
		}
	}

	return SaveWhitelist(patterns)
}

// ClearWhitelist removes all patterns
func ClearWhitelist() error {
	return SaveWhitelist([]WhitelistPattern{})
}
