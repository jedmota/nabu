package viewmodel

import (
	"proxy-tui/internal/config"
	"proxy-tui/internal/model"
)

// GetAlertRules returns the current alert rules.
func (vm *ViewModel) GetAlertRules() []model.AlertRule {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	result := make([]model.AlertRule, len(vm.alertRules))
	copy(result, vm.alertRules)
	return result
}

// SetAlertRules replaces the alert rules and persists them.
func (vm *ViewModel) SetAlertRules(rules []model.AlertRule) {
	vm.mu.Lock()
	vm.alertRules = rules
	vm.mu.Unlock()
	config.SaveAlerts(rules)
}

// ToggleAlertRule toggles the enabled state of an alert rule by index.
func (vm *ViewModel) ToggleAlertRule(index int) {
	vm.mu.Lock()
	if index >= 0 && index < len(vm.alertRules) {
		vm.alertRules[index].Enabled = !vm.alertRules[index].Enabled
	}
	rules := make([]model.AlertRule, len(vm.alertRules))
	copy(rules, vm.alertRules)
	vm.mu.Unlock()
	config.SaveAlerts(rules)
}

// CheckAlerts returns true if any enabled alert rule matches the flow.
func (vm *ViewModel) CheckAlerts(flow *model.Flow) bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	for i := range vm.alertRules {
		if vm.alertRules[i].Match(flow) {
			return true
		}
	}
	return false
}
