package viewmodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

	var sb strings.Builder

	// Request section
	sb.WriteString("[yellow]═══ Request ═══[-]\n\n")
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
