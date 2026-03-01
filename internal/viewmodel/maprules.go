package viewmodel

import (
	"strings"

	"nabu/internal/config"
	"nabu/internal/model"
)

// AddMapLocalRule adds a map local rule
func (vm *ViewModel) AddMapLocalRule(pattern, localPath string, statusCode int, contentType, method string) {
	rule := model.NewMapLocalRule(pattern, localPath, statusCode, contentType, method)
	vm.source.MapRules().Add(rule)

	// Save to config
	vm.config.AddMapLocalEntry(config.MapLocalEntry{
		Pattern:     pattern,
		LocalPath:   localPath,
		Enabled:     true,
		StatusCode:  statusCode,
		ContentType: contentType,
		Method:      strings.ToUpper(method),
	})
	vm.notifyConfigChange()
}

// RemoveMapLocalRule removes a map local rule by ID
func (vm *ViewModel) RemoveMapLocalRule(id int) {
	rule := vm.source.MapRules().GetByID(id)
	if rule != nil {
		vm.config.RemoveMapLocalEntry(rule.Pattern)
	}
	vm.source.MapRules().Remove(id)
	vm.notifyConfigChange()
}

// ToggleMapLocalRule toggles a map local rule
func (vm *ViewModel) ToggleMapLocalRule(id int) {
	rule := vm.source.MapRules().GetByID(id)
	if rule != nil {
		vm.config.ToggleMapLocalEntry(rule.Pattern)
	}
	vm.source.MapRules().Toggle(id)
	vm.notifyConfigChange()
}

// GetMapLocalRules returns all map local rules
func (vm *ViewModel) GetMapLocalRules() []*model.MapRule {
	rules := vm.source.MapRules().All()
	localRules := make([]*model.MapRule, 0)
	for _, r := range rules {
		if r.Type == model.MapLocal {
			localRules = append(localRules, r)
		}
	}
	return localRules
}

// LoadMapLocalRules loads map local rules from config
func (vm *ViewModel) LoadMapLocalRules() {
	entries, err := vm.config.LoadMapLocal()
	if err != nil {
		return
	}

	for _, e := range entries {
		rule := model.NewMapLocalRule(e.Pattern, e.LocalPath, e.StatusCode, e.ContentType, e.Method)
		rule.Enabled = e.Enabled
		vm.source.MapRules().Add(rule)
	}
}

// AddMapRemoteRule adds a map remote rule
func (vm *ViewModel) AddMapRemoteRule(pattern, remoteURL, method string) {
	rule := model.NewMapRemoteRule(pattern, remoteURL, method)
	vm.source.MapRules().Add(rule)

	// Save to config
	vm.config.AddMapRemoteEntry(config.MapRemoteEntry{
		Pattern:   pattern,
		RemoteURL: remoteURL,
		Enabled:   true,
		Method:    strings.ToUpper(method),
	})
	vm.notifyConfigChange()
}

// RemoveMapRemoteRule removes a map remote rule by ID
func (vm *ViewModel) RemoveMapRemoteRule(id int) {
	rule := vm.source.MapRules().GetByID(id)
	if rule != nil {
		vm.config.RemoveMapRemoteEntry(rule.Pattern)
	}
	vm.source.MapRules().Remove(id)
	vm.notifyConfigChange()
}

// ToggleMapRemoteRule toggles a map remote rule
func (vm *ViewModel) ToggleMapRemoteRule(id int) {
	rule := vm.source.MapRules().GetByID(id)
	if rule != nil {
		vm.config.ToggleMapRemoteEntry(rule.Pattern)
	}
	vm.source.MapRules().Toggle(id)
	vm.notifyConfigChange()
}

// GetMapRemoteRules returns all map remote rules
func (vm *ViewModel) GetMapRemoteRules() []*model.MapRule {
	rules := vm.source.MapRules().All()
	remoteRules := make([]*model.MapRule, 0)
	for _, r := range rules {
		if r.Type == model.MapRemote {
			remoteRules = append(remoteRules, r)
		}
	}
	return remoteRules
}

// GetMapRemoteRuleByID returns a map remote rule by ID
func (vm *ViewModel) GetMapRemoteRuleByID(id int) *model.MapRule {
	return vm.source.MapRules().GetByID(id)
}

// UpdateMapRemoteRule updates an existing map remote rule
func (vm *ViewModel) UpdateMapRemoteRule(id int, pattern, remoteURL, method string) {
	oldRule := vm.source.MapRules().GetByID(id)
	if oldRule == nil {
		return
	}

	oldPattern := oldRule.Pattern
	enabled := oldRule.Enabled

	// Create a new rule to ensure pattern is properly compiled
	newRule := model.NewMapRemoteRule(pattern, remoteURL, method)
	newRule.ID = id
	newRule.Enabled = enabled
	vm.source.MapRules().Update(newRule)

	// Update config
	vm.config.UpdateMapRemoteEntry(oldPattern, config.MapRemoteEntry{
		Pattern:   pattern,
		RemoteURL: remoteURL,
		Enabled:   enabled,
		Method:    strings.ToUpper(method),
	})
	vm.notifyConfigChange()
}

// LoadMapRemoteRules loads map remote rules from config
func (vm *ViewModel) LoadMapRemoteRules() {
	entries, err := vm.config.LoadMapRemote()
	if err != nil {
		return
	}

	for _, e := range entries {
		rule := model.NewMapRemoteRule(e.Pattern, e.RemoteURL, e.Method)
		rule.Enabled = e.Enabled
		vm.source.MapRules().Add(rule)
	}
}
