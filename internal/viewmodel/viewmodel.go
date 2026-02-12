package viewmodel

import (
	"sync"

	"proxy-tui/internal/config"
	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

// ConfigPersistence abstracts config file operations so the ViewModel
// can be tested without touching the filesystem.
type ConfigPersistence interface {
	// Whitelist
	LoadWhitelist() ([]config.WhitelistPattern, error)
	AddToWhitelist(pattern string) error
	RemoveFromWhitelist(pattern string) error
	EditWhitelistPattern(oldPattern, newPattern string) error
	ToggleWhitelistPattern(pattern string) error
	ClearWhitelist() error

	// Map-local
	LoadMapLocal() ([]config.MapLocalEntry, error)
	AddMapLocalEntry(entry config.MapLocalEntry) error
	RemoveMapLocalEntry(pattern string) error
	ToggleMapLocalEntry(pattern string) error

	// Map-remote
	LoadMapRemote() ([]config.MapRemoteEntry, error)
	AddMapRemoteEntry(entry config.MapRemoteEntry) error
	RemoveMapRemoteEntry(pattern string) error
	ToggleMapRemoteEntry(pattern string) error
	UpdateMapRemoteEntry(oldPattern string, entry config.MapRemoteEntry) error
}

// ViewModel mediates between the proxy and the UI
type ViewModel struct {
	source        proxy.FlowSource
	flowStore     *proxy.FlowStore
	config        ConfigPersistence
	filteredFlows []*model.Flow
	selectedFlow  *model.Flow
	selectedIndex int
	filter        *model.FilterState
	alertRules    []model.AlertRule
	mu            sync.RWMutex
	updateChan    chan struct{}
	secondary     bool
}

// New creates a new ViewModel
func New(source proxy.FlowSource, cfg ConfigPersistence) *ViewModel {
	vm := &ViewModel{
		source:        source,
		flowStore:     source.FlowStore(),
		config:        cfg,
		filteredFlows: make([]*model.Flow, 0),
		filter:        model.NewFilterState(),
		updateChan:    make(chan struct{}, 100),
	}

	// Load saved whitelist patterns
	if patterns, err := vm.config.LoadWhitelist(); err == nil {
		for _, p := range patterns {
			vm.filter.HostPatterns = append(vm.filter.HostPatterns, model.HostPattern{
				Pattern: p.Pattern,
				Enabled: p.Enabled,
			})
			// Only add enabled patterns to SSL proxy list
			if p.Enabled {
				vm.source.SSLProxyList().Add(p.Pattern)
			}
		}
	}

	// Load saved map local rules
	vm.LoadMapLocalRules()

	// Load saved map remote rules
	vm.LoadMapRemoteRules()

	// Load alert rules
	if rules, err := config.LoadAlerts(); err == nil {
		vm.alertRules = rules
	} else {
		vm.alertRules = model.DefaultAlertRules()
	}

	return vm
}

// StartEventLoop starts listening for flow events
func (vm *ViewModel) StartEventLoop() {
	go func() {
		for range vm.source.Events() {
			vm.applyFilter()
			// Signal update
			select {
			case vm.updateChan <- struct{}{}:
			default:
			}
		}
	}()
}

// Updates returns the update notification channel
func (vm *ViewModel) Updates() <-chan struct{} {
	return vm.updateChan
}

// applyFilter filters flows based on current filter state
func (vm *ViewModel) applyFilter() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.filteredFlows = vm.flowStore.Filter(vm.filter)
}

// GetFilteredFlows returns the current filtered flows
func (vm *ViewModel) GetFilteredFlows() []*model.Flow {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	result := make([]*model.Flow, len(vm.filteredFlows))
	copy(result, vm.filteredFlows)
	return result
}

// GetFlowCount returns the total number of flows
func (vm *ViewModel) GetFlowCount() int {
	return vm.flowStore.Count()
}

// GetFilteredCount returns the number of filtered flows
func (vm *ViewModel) GetFilteredCount() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return len(vm.filteredFlows)
}

// SelectFlow selects a flow by index
func (vm *ViewModel) SelectFlow(index int) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if index < 0 || index >= len(vm.filteredFlows) {
		vm.selectedFlow = nil
		vm.selectedIndex = -1
		return
	}

	vm.selectedIndex = index
	vm.selectedFlow = vm.filteredFlows[index]
}

// GetSelectedFlow returns the currently selected flow
func (vm *ViewModel) GetSelectedFlow() *model.Flow {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.selectedFlow
}

// GetSelectedIndex returns the current selection index
func (vm *ViewModel) GetSelectedIndex() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.selectedIndex
}

// SetSearchQuery sets the search query and refilters
func (vm *ViewModel) SetSearchQuery(query string) {
	vm.mu.Lock()
	vm.filter.SearchQuery = query
	vm.mu.Unlock()
	vm.applyFilter()
}

// SetFilterType sets the filter type and refilters
func (vm *ViewModel) SetFilterType(filterType model.FilterType) {
	vm.mu.Lock()
	vm.filter.Type = filterType
	vm.mu.Unlock()
	vm.applyFilter()
}

// GetFilter returns the current filter state
func (vm *ViewModel) GetFilter() *model.FilterState {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.filter
}

// Refresh forces a refresh of the filtered flows
func (vm *ViewModel) Refresh() {
	vm.applyFilter()
	select {
	case vm.updateChan <- struct{}{}:
	default:
	}
}

// ImportFlows adds the given flows to the flow store and refreshes.
func (vm *ViewModel) ImportFlows(flows []*model.Flow) int {
	count := 0
	for _, f := range flows {
		if f != nil && f.Request != nil {
			vm.flowStore.AddDirect(f)
			count++
		}
	}
	vm.Refresh()
	return count
}

// TogglePause toggles the paused state. Returns true if now paused.
func (vm *ViewModel) TogglePause() bool {
	paused := !vm.flowStore.IsPaused()
	vm.flowStore.SetPaused(paused)
	return paused
}

// IsPaused returns whether capture is paused.
func (vm *ViewModel) IsPaused() bool {
	return vm.flowStore.IsPaused()
}

// ToggleStar toggles the starred state of a flow. Returns true if now starred.
func (vm *ViewModel) ToggleStar(flow *model.Flow) bool {
	if flow == nil {
		return false
	}
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if vm.filter.StarredIDs[flow.ID] {
		delete(vm.filter.StarredIDs, flow.ID)
		return false
	}
	vm.filter.StarredIDs[flow.ID] = true
	return true
}

// StarFlows stars all the given flows.
func (vm *ViewModel) StarFlows(flows []*model.Flow) int {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	count := 0
	for _, f := range flows {
		if f != nil && !vm.filter.StarredIDs[f.ID] {
			vm.filter.StarredIDs[f.ID] = true
			count++
		}
	}
	return count
}

// IsStarred returns whether a flow is starred.
func (vm *ViewModel) IsStarred(flow *model.Flow) bool {
	if flow == nil {
		return false
	}
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.filter.StarredIDs[flow.ID]
}

// StarredCount returns the number of starred flows.
func (vm *ViewModel) StarredCount() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return len(vm.filter.StarredIDs)
}

// ClearFlows clears all flows
func (vm *ViewModel) ClearFlows() {
	vm.flowStore.Clear()
	vm.mu.Lock()
	vm.filteredFlows = make([]*model.Flow, 0)
	vm.selectedFlow = nil
	vm.selectedIndex = -1
	vm.mu.Unlock()
	vm.Refresh()
}

// Port returns the port number from the flow source.
func (vm *ViewModel) Port() int {
	return vm.source.Port()
}

// BindAddress returns the bind address from the flow source.
func (vm *ViewModel) BindAddress() string {
	return vm.source.BindAddress()
}

// SetSecondary marks this ViewModel as running in secondary (IPC client) mode.
func (vm *ViewModel) SetSecondary(v bool) {
	vm.secondary = v
}

// IsSecondary returns true if this ViewModel is running as a secondary instance.
func (vm *ViewModel) IsSecondary() bool {
	return vm.secondary
}

// ConfigNotifier is implemented by FlowSource adapters that can notify a
// primary instance to reload its configuration (e.g. the IPC Adapter).
type ConfigNotifier interface {
	NotifyConfigChange()
}

// notifyConfigChange tells the primary to reload if we are a secondary instance.
func (vm *ViewModel) notifyConfigChange() {
	if !vm.secondary {
		return
	}
	if n, ok := vm.source.(ConfigNotifier); ok {
		n.NotifyConfigChange()
	}
}

// ReloadConfig reloads all configuration (whitelist, map-local, map-remote)
// from disk and replaces the in-memory state.
func (vm *ViewModel) ReloadConfig() {
	// Reload whitelist
	vm.mu.Lock()
	vm.filter.HostPatterns = nil
	vm.mu.Unlock()
	vm.source.SSLProxyList().Clear()

	if patterns, err := vm.config.LoadWhitelist(); err == nil {
		vm.mu.Lock()
		for _, p := range patterns {
			vm.filter.HostPatterns = append(vm.filter.HostPatterns, model.HostPattern{
				Pattern: p.Pattern,
				Enabled: p.Enabled,
			})
		}
		vm.mu.Unlock()
		for _, p := range patterns {
			if p.Enabled {
				vm.source.SSLProxyList().Add(p.Pattern)
			}
		}
	}

	// Reload map rules
	vm.source.MapRules().Clear()
	vm.LoadMapLocalRules()
	vm.LoadMapRemoteRules()

	vm.applyFilter()
	vm.Refresh()
}
