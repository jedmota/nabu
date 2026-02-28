package viewmodel

import (
	"nabu/internal/model"
)

// AddWhitelistPattern adds a pattern to the whitelist and SSL proxy list
func (vm *ViewModel) AddWhitelistPattern(pattern string) {
	if pattern == "" {
		return
	}
	vm.mu.Lock()
	// Check if pattern already exists
	for _, hp := range vm.filter.HostPatterns {
		if hp.Pattern == pattern {
			vm.mu.Unlock()
			return
		}
	}
	vm.filter.HostPatterns = append(vm.filter.HostPatterns, model.HostPattern{
		Pattern: pattern,
		Enabled: true,
	})
	vm.mu.Unlock()

	// Also add to proxy's SSL proxy list for MITM
	vm.source.SSLProxyList().Add(pattern)

	// Save to config file
	vm.config.AddToWhitelist(pattern)

	vm.notifyConfigChange()
	vm.applyFilter()
}

// RemoveWhitelistPattern removes a pattern from the whitelist
func (vm *ViewModel) RemoveWhitelistPattern(pattern string) {
	vm.mu.Lock()
	for i, hp := range vm.filter.HostPatterns {
		if hp.Pattern == pattern {
			vm.filter.HostPatterns = append(vm.filter.HostPatterns[:i], vm.filter.HostPatterns[i+1:]...)
			break
		}
	}
	vm.mu.Unlock()

	// Also remove from proxy's SSL proxy list
	vm.source.SSLProxyList().Remove(pattern)

	// Save to config file
	vm.config.RemoveFromWhitelist(pattern)

	vm.notifyConfigChange()
	vm.applyFilter()
}

// EditWhitelistPattern replaces an old pattern with a new one, preserving enabled state
func (vm *ViewModel) EditWhitelistPattern(oldPattern, newPattern string) {
	if newPattern == "" || oldPattern == newPattern {
		return
	}
	vm.mu.Lock()
	for i, hp := range vm.filter.HostPatterns {
		if hp.Pattern == oldPattern {
			vm.filter.HostPatterns[i].Pattern = newPattern
			// Update proxy's SSL proxy list
			if hp.Enabled {
				vm.source.SSLProxyList().Remove(oldPattern)
				vm.source.SSLProxyList().Add(newPattern)
			}
			break
		}
	}
	vm.mu.Unlock()

	// Update config
	vm.config.EditWhitelistPattern(oldPattern, newPattern)

	vm.notifyConfigChange()
	vm.applyFilter()
}

// ToggleWhitelistPattern toggles the enabled state of a pattern
func (vm *ViewModel) ToggleWhitelistPattern(pattern string) {
	vm.mu.Lock()
	var enabled bool
	for i, hp := range vm.filter.HostPatterns {
		if hp.Pattern == pattern {
			vm.filter.HostPatterns[i].Enabled = !hp.Enabled
			enabled = vm.filter.HostPatterns[i].Enabled
			break
		}
	}
	vm.mu.Unlock()

	// Update proxy's SSL proxy list
	if enabled {
		vm.source.SSLProxyList().Add(pattern)
	} else {
		vm.source.SSLProxyList().Remove(pattern)
	}

	// Save to config file
	vm.config.ToggleWhitelistPattern(pattern)

	vm.notifyConfigChange()
	vm.applyFilter()
}

// GetWhitelistPatterns returns the current whitelist patterns with their enabled state
func (vm *ViewModel) GetWhitelistPatterns() []model.HostPattern {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	result := make([]model.HostPattern, len(vm.filter.HostPatterns))
	copy(result, vm.filter.HostPatterns)
	return result
}

// ClearWhitelist removes all whitelist patterns
func (vm *ViewModel) ClearWhitelist() {
	vm.mu.Lock()
	vm.filter.HostPatterns = []model.HostPattern{}
	vm.mu.Unlock()

	// Also clear proxy's SSL proxy list
	vm.source.SSLProxyList().Clear()

	// Save to config file
	vm.config.ClearWhitelist()

	vm.notifyConfigChange()
	vm.applyFilter()
}
