package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"nabu/internal/model"
)

const alertsFile = "alerts.json"

// LoadAlerts loads alert rules from the config file.
// Returns default rules if the file doesn't exist.
func LoadAlerts() ([]model.AlertRule, error) {
	path := filepath.Join(GetConfigDir(), alertsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultAlertRules(), nil
		}
		return nil, err
	}

	var rules []model.AlertRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return model.DefaultAlertRules(), nil
	}
	return rules, nil
}

// SaveAlerts saves alert rules to the config file.
func SaveAlerts(rules []model.AlertRule) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, alertsFile), data, 0644)
}
