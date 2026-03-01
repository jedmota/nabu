package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"nabu/internal/util"
)

// ConfigEntry is the constraint for entries stored by ConfigStore.
type ConfigEntry interface {
	GetPattern() string
	GetEnabled() bool
}

// ConfigStore provides generic Load/Save/Add/Remove/Toggle/Update for
// JSONC-backed config files whose structure is { "<wrapKey>": [entries...] }.
type ConfigStore[E ConfigEntry] struct {
	filename string
	header   []string // JSONC comment lines written at the top
	wrapKey  string   // JSON key wrapping the entry array
	toggleFn func(E) E
}

// Load reads and parses the config file, returning the entry slice.
func (s *ConfigStore[E]) Load() ([]E, error) {
	path := filepath.Join(GetConfigDir(), s.filename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []E{}, nil
		}
		return nil, err
	}

	jsonData := util.StripJSONComments(data)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return []E{}, nil
	}

	entriesJSON, ok := raw[s.wrapKey]
	if !ok {
		return []E{}, nil
	}

	var entries []E
	if err := json.Unmarshal(entriesJSON, &entries); err != nil {
		return []E{}, nil
	}

	return entries, nil
}

// Save writes entries to the config file in JSONC format.
func (s *ConfigStore[E]) Save(entries []E) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, s.filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("{\n")
	for _, h := range s.header {
		file.WriteString("  " + h + "\n")
	}
	file.WriteString("  \"" + s.wrapKey + "\": [\n")

	for i, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		comma := ","
		if i == len(entries)-1 {
			comma = ""
		}
		file.WriteString("    " + string(data) + comma + "\n")
	}

	file.WriteString("  ]\n")
	file.WriteString("}\n")

	return nil
}

// Add appends an entry (no-op if pattern already exists) and saves.
func (s *ConfigStore[E]) Add(entry E) error {
	entries, err := s.Load()
	if err != nil {
		entries = []E{}
	}

	for _, e := range entries {
		if e.GetPattern() == entry.GetPattern() {
			return nil
		}
	}

	entries = append(entries, entry)
	return s.Save(entries)
}

// Remove deletes the entry matching pattern and saves.
func (s *ConfigStore[E]) Remove(pattern string) error {
	entries, err := s.Load()
	if err != nil {
		return err
	}

	var filtered []E
	for _, e := range entries {
		if e.GetPattern() != pattern {
			filtered = append(filtered, e)
		}
	}

	return s.Save(filtered)
}

// Toggle flips the enabled state of the entry matching pattern.
func (s *ConfigStore[E]) Toggle(pattern string) error {
	entries, err := s.Load()
	if err != nil {
		return err
	}

	for i, e := range entries {
		if e.GetPattern() == pattern {
			entries[i] = s.toggleFn(e)
			break
		}
	}

	return s.Save(entries)
}

// Update replaces the entry matching oldPattern with the new entry.
func (s *ConfigStore[E]) Update(oldPattern string, entry E) error {
	entries, err := s.Load()
	if err != nil {
		return err
	}

	for i, e := range entries {
		if e.GetPattern() == oldPattern {
			entries[i] = entry
			break
		}
	}

	return s.Save(entries)
}
