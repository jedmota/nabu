package viewmodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"proxy-tui/internal/config"
	"proxy-tui/internal/model"
	"proxy-tui/internal/proxy"
)

// ViewModel mediates between the proxy and the UI
type ViewModel struct {
	proxy         *proxy.Proxy
	flowStore     *proxy.FlowStore
	filteredFlows []*model.Flow
	selectedFlow  *model.Flow
	selectedIndex int
	filter        *model.FilterState
	mu            sync.RWMutex
	updateChan    chan struct{}
}

// New creates a new ViewModel
func New(p *proxy.Proxy) *ViewModel {
	vm := &ViewModel{
		proxy:         p,
		flowStore:     p.FlowStore(),
		filteredFlows: make([]*model.Flow, 0),
		filter:        model.NewFilterState(),
		updateChan:    make(chan struct{}, 100),
	}

	// Load saved whitelist patterns
	if patterns, err := config.LoadWhitelist(); err == nil {
		for _, p := range patterns {
			vm.filter.HostPatterns = append(vm.filter.HostPatterns, model.HostPattern{
				Pattern: p.Pattern,
				Enabled: p.Enabled,
			})
			// Only add enabled patterns to SSL proxy list
			if p.Enabled {
				vm.proxy.SSLProxyList().Add(p.Pattern)
			}
		}
	}

	// Load saved map local rules
	vm.LoadMapLocalRules()

	// Load saved map remote rules
	vm.LoadMapRemoteRules()

	return vm
}

// StartEventLoop starts listening for flow events
func (vm *ViewModel) StartEventLoop() {
	go func() {
		for range vm.proxy.Events() {
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
	vm.proxy.SSLProxyList().Add(pattern)

	// Save to config file
	config.AddToWhitelist(pattern)

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
	vm.proxy.SSLProxyList().Remove(pattern)

	// Save to config file
	config.RemoveFromWhitelist(pattern)

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
		vm.proxy.SSLProxyList().Add(pattern)
	} else {
		vm.proxy.SSLProxyList().Remove(pattern)
	}

	// Save to config file
	config.ToggleWhitelistPattern(pattern)

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
	vm.proxy.SSLProxyList().Clear()

	// Save to config file
	config.ClearWhitelist()

	vm.applyFilter()
}

// Refresh forces a refresh of the filtered flows
func (vm *ViewModel) Refresh() {
	vm.applyFilter()
	select {
	case vm.updateChan <- struct{}{}:
	default:
	}
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

// FormatFlowSummary returns a formatted summary of a flow for the list
func (vm *ViewModel) FormatFlowSummary(flow *model.Flow) (method, host, path, status, duration string) {
	if flow == nil || flow.Request == nil {
		return "", "", "", "", ""
	}

	method = flow.Request.Method
	host = flow.Request.Host
	path = flow.Request.Path
	if path == "" {
		path = "/"
	}

	if flow.Response != nil {
		status = fmt.Sprintf("%d", flow.Response.StatusCode)
	} else if flow.Error != nil {
		status = "ERR"
	} else {
		status = "..."
	}

	if flow.IsComplete() {
		duration = formatDuration(flow.Duration())
	} else {
		duration = "..."
	}

	return
}

// FormatFlowDetail returns a detailed formatted view of a flow
func (vm *ViewModel) FormatFlowDetail(flow *model.Flow, raw bool) string {
	if flow == nil || flow.Request == nil {
		return "No flow selected"
	}

	// Handle tunneled flows
	if flow.Tunneled {
		var sb strings.Builder
		sb.WriteString("[yellow]═══ Tunneled Connection ═══[-]\n\n")
		sb.WriteString(fmt.Sprintf("[gray]CONNECT[-] %s\n\n", flow.Request.Host))
		sb.WriteString("[gray]This connection was tunneled without SSL interception.[-]\n")
		sb.WriteString("[gray]The request and response content is encrypted.[-]\n\n")
		sb.WriteString("To inspect this traffic, add the host to the whitelist:\n")
		sb.WriteString(fmt.Sprintf("  Press [yellow]w[-] and enter: [green]%s[-]\n", flow.Request.Host))
		sb.WriteString(fmt.Sprintf("  Or use wildcard: [green]*.%s[-]\n", getBaseDomain(flow.Request.Host)))
		return sb.String()
	}

	var sb strings.Builder

	// Request section
	sb.WriteString("[yellow]═══ Request ═══[-]\n\n")
	sb.WriteString(fmt.Sprintf("[gray]%s[-]\n", flow.StartTime.Format("2006-01-02 15:04:05.000")))
	sb.WriteString(fmt.Sprintf("[green]%s[-] %s %s\n", flow.Request.Method, flow.Request.URL, flow.Request.Proto))
	sb.WriteString(fmt.Sprintf("Host: %s\n", flow.Request.Host))

	// Request headers
	sb.WriteString("\n[blue]Headers:[-]\n")
	for key, values := range flow.Request.Headers {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// Request body
	if len(flow.Request.Body) > 0 {
		sb.WriteString("\n[blue]Body:[-]\n")
		if raw {
			sb.WriteString(string(flow.Request.Body))
		} else {
			sb.WriteString(formatBody(flow.Request.Body, flow.Request.Headers.Get("Content-Type")))
		}
		sb.WriteString("\n")
	}

	// Response section
	if flow.Response != nil {
		sb.WriteString("\n[yellow]═══ Response ═══[-]\n\n")
		sb.WriteString(fmt.Sprintf("[cyan]%s[-] %s\n", flow.Response.Status, flow.Response.Proto))

		// Response headers
		sb.WriteString("\n[blue]Headers:[-]\n")
		for key, values := range flow.Response.Headers {
			for _, value := range values {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}

		// Response body
		if len(flow.Response.Body) > 0 {
			sb.WriteString("\n[blue]Body:[-]\n")
			if raw {
				sb.WriteString(string(flow.Response.Body))
			} else {
				sb.WriteString(formatBody(flow.Response.Body, flow.Response.Headers.Get("Content-Type")))
			}
			sb.WriteString("\n")
		}
	} else if flow.Error != nil {
		sb.WriteString("\n[red]═══ Error ═══[-]\n\n")
		sb.WriteString(flow.Error.Error())
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n[gray]Waiting for response...[-]\n")
	}

	// Timing info
	sb.WriteString("\n[yellow]═══ Timing ═══[-]\n\n")
	sb.WriteString(fmt.Sprintf("Started: %s\n", flow.StartTime.Format("15:04:05.000")))
	if flow.IsComplete() {
		sb.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(flow.Duration())))
	}

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d interface{ Milliseconds() int64 }) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000)
}

// formatBody formats the body based on content type
func formatBody(body []byte, contentType string) string {
	maxLen := 10000

	// Try to pretty-print JSON
	if strings.Contains(contentType, "json") || isJSON(body) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			result := prettyJSON.String()
			if len(result) > maxLen {
				return result[:maxLen] + "\n... (truncated)"
			}
			return result
		}
	}

	// Default: return as string
	if len(body) > maxLen {
		return string(body[:maxLen]) + "\n... (truncated)"
	}
	return string(body)
}

// isJSON checks if the body looks like JSON
func isJSON(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	// Trim whitespace and check first character
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}

// GetProxy returns the underlying proxy
func (vm *ViewModel) GetProxy() *proxy.Proxy {
	return vm.proxy
}

// getBaseDomain extracts the base domain from a host (e.g., "api.example.com" -> "example.com")
func getBaseDomain(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	// Return last two parts
	return strings.Join(parts[len(parts)-2:], ".")
}

// AddMapLocalRule adds a map local rule
func (vm *ViewModel) AddMapLocalRule(pattern, localPath string, statusCode int, contentType string) {
	rule := model.NewMapLocalRule(pattern, localPath, statusCode, contentType)
	vm.proxy.MapRules().Add(rule)

	// Save to config
	config.AddMapLocalEntry(config.MapLocalEntry{
		Pattern:     pattern,
		LocalPath:   localPath,
		Enabled:     true,
		StatusCode:  statusCode,
		ContentType: contentType,
	})
}

// RemoveMapLocalRule removes a map local rule by ID
func (vm *ViewModel) RemoveMapLocalRule(id int) {
	rule := vm.proxy.MapRules().GetByID(id)
	if rule != nil {
		config.RemoveMapLocalEntry(rule.Pattern)
	}
	vm.proxy.MapRules().Remove(id)
}

// ToggleMapLocalRule toggles a map local rule
func (vm *ViewModel) ToggleMapLocalRule(id int) {
	rule := vm.proxy.MapRules().GetByID(id)
	if rule != nil {
		config.ToggleMapLocalEntry(rule.Pattern)
	}
	vm.proxy.MapRules().Toggle(id)
}

// GetMapLocalRules returns all map local rules
func (vm *ViewModel) GetMapLocalRules() []*model.MapRule {
	rules := vm.proxy.MapRules().All()
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
	entries, err := config.LoadMapLocal()
	if err != nil {
		return
	}

	for _, e := range entries {
		rule := model.NewMapLocalRule(e.Pattern, e.LocalPath, e.StatusCode, e.ContentType)
		rule.Enabled = e.Enabled
		vm.proxy.MapRules().Add(rule)
	}
}

// AddMapRemoteRule adds a map remote rule
func (vm *ViewModel) AddMapRemoteRule(pattern, remoteURL string) {
	rule := model.NewMapRemoteRule(pattern, remoteURL)
	vm.proxy.MapRules().Add(rule)

	// Save to config
	config.AddMapRemoteEntry(config.MapRemoteEntry{
		Pattern:   pattern,
		RemoteURL: remoteURL,
		Enabled:   true,
	})
}

// RemoveMapRemoteRule removes a map remote rule by ID
func (vm *ViewModel) RemoveMapRemoteRule(id int) {
	rule := vm.proxy.MapRules().GetByID(id)
	if rule != nil {
		config.RemoveMapRemoteEntry(rule.Pattern)
	}
	vm.proxy.MapRules().Remove(id)
}

// ToggleMapRemoteRule toggles a map remote rule
func (vm *ViewModel) ToggleMapRemoteRule(id int) {
	rule := vm.proxy.MapRules().GetByID(id)
	if rule != nil {
		config.ToggleMapRemoteEntry(rule.Pattern)
	}
	vm.proxy.MapRules().Toggle(id)
}

// GetMapRemoteRules returns all map remote rules
func (vm *ViewModel) GetMapRemoteRules() []*model.MapRule {
	rules := vm.proxy.MapRules().All()
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
	return vm.proxy.MapRules().GetByID(id)
}

// UpdateMapRemoteRule updates an existing map remote rule
func (vm *ViewModel) UpdateMapRemoteRule(id int, pattern, remoteURL string) {
	oldRule := vm.proxy.MapRules().GetByID(id)
	if oldRule == nil {
		return
	}

	oldPattern := oldRule.Pattern
	enabled := oldRule.Enabled

	// Create a new rule to ensure pattern is properly compiled
	newRule := model.NewMapRemoteRule(pattern, remoteURL)
	newRule.ID = id
	newRule.Enabled = enabled
	vm.proxy.MapRules().Update(newRule)

	// Update config
	config.UpdateMapRemoteEntry(oldPattern, config.MapRemoteEntry{
		Pattern:   pattern,
		RemoteURL: remoteURL,
		Enabled:   enabled,
	})
}

// LoadMapRemoteRules loads map remote rules from config
func (vm *ViewModel) LoadMapRemoteRules() {
	entries, err := config.LoadMapRemote()
	if err != nil {
		return
	}

	for _, e := range entries {
		rule := model.NewMapRemoteRule(e.Pattern, e.RemoteURL)
		rule.Enabled = e.Enabled
		vm.proxy.MapRules().Add(rule)
	}
}
