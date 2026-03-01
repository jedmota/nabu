package viewmodel

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"nabu/internal/config"
	"nabu/internal/model"
	"nabu/internal/proxy"
)

// mockFlowSource implements proxy.FlowSource for testing.
type mockFlowSource struct {
	eventCh      chan model.FlowEvent
	flowStore    *proxy.FlowStore
	sslProxyList *proxy.SSLProxyList
	mapRules     *model.MapRuleStore
	port         int
	bindAddress  string
}

func newMockFlowSource() *mockFlowSource {
	ch := make(chan model.FlowEvent, 100)
	return &mockFlowSource{
		eventCh:      ch,
		flowStore:    proxy.NewFlowStore(1000, ch),
		sslProxyList: proxy.NewSSLProxyList(),
		mapRules:     model.NewMapRuleStore(),
		port:         9090,
		bindAddress:  "0.0.0.0",
	}
}

func (m *mockFlowSource) Events() <-chan model.FlowEvent    { return m.eventCh }
func (m *mockFlowSource) FlowStore() *proxy.FlowStore       { return m.flowStore }
func (m *mockFlowSource) SSLProxyList() *proxy.SSLProxyList  { return m.sslProxyList }
func (m *mockFlowSource) MapRules() *model.MapRuleStore      { return m.mapRules }
func (m *mockFlowSource) Port() int                          { return m.port }
func (m *mockFlowSource) BindAddress() string                { return m.bindAddress }

// newTestVM creates a ViewModel with a mock source and temp HOME
// so config loading doesn't touch real files.
func newTestVM(t *testing.T) (*ViewModel, *mockFlowSource) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	src := newMockFlowSource()
	vm := New(src, config.DefaultPersistence{})
	return vm, src
}

// --- Constructor ---

func TestNew_InitialState(t *testing.T) {
	vm, _ := newTestVM(t)

	if len(vm.GetFilteredFlows()) != 0 {
		t.Error("initial flows should be empty")
	}
	if vm.GetSelectedFlow() != nil {
		t.Error("initial selection should be nil")
	}
}

// --- FormatFlowSummary ---

func TestFormatFlowSummary_Complete(t *testing.T) {
	vm, _ := newTestVM(t)

	flow := &model.Flow{
		StartTime: time.Now().Add(-100 * time.Millisecond),
		EndTime:   time.Now(),
		Request: &model.Request{
			Method: "GET",
			Host:   "example.com",
			Path:   "/api",
		},
		Response: &model.Response{StatusCode: 200},
	}

	method, host, path, status, dur := vm.FormatFlowSummary(flow)
	if method != "GET" {
		t.Errorf("method = %q", method)
	}
	if host != "example.com" {
		t.Errorf("host = %q", host)
	}
	if path != "/api" {
		t.Errorf("path = %q", path)
	}
	if status != "200" {
		t.Errorf("status = %q", status)
	}
	if dur == "" || dur == "..." {
		t.Errorf("duration = %q, want formatted", dur)
	}
}

func TestFormatFlowSummary_Nil(t *testing.T) {
	vm, _ := newTestVM(t)

	method, host, path, status, dur := vm.FormatFlowSummary(nil)
	if method != "" || host != "" || path != "" || status != "" || dur != "" {
		t.Error("nil flow should return all empty strings")
	}
}

func TestFormatFlowSummary_NoResponse(t *testing.T) {
	vm, _ := newTestVM(t)

	flow := &model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", Host: "h", Path: "/"},
	}
	_, _, _, status, _ := vm.FormatFlowSummary(flow)
	if status != "..." {
		t.Errorf("status = %q, want '...'", status)
	}
}

func TestFormatFlowSummary_Error(t *testing.T) {
	vm, _ := newTestVM(t)

	flow := &model.Flow{
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Request:   &model.Request{Method: "GET", Host: "h", Path: "/"},
		Error:     errors.New("fail"),
	}
	_, _, _, status, _ := vm.FormatFlowSummary(flow)
	if status != "ERR" {
		t.Errorf("status = %q, want 'ERR'", status)
	}
}

// --- FormatFlowDetail ---

func TestFormatFlowDetail_Tunneled(t *testing.T) {
	vm, _ := newTestVM(t)

	flow := &model.Flow{
		StartTime: time.Now(),
		Tunneled:  true,
		Request:   &model.Request{Method: "CONNECT", Host: "example.com"},
	}
	detail := vm.FormatFlowDetail(flow, false)
	if detail == "" {
		t.Error("should produce output for tunneled flow")
	}
}

func TestFormatFlowDetail_WithResponse(t *testing.T) {
	vm, _ := newTestVM(t)

	flow := &model.Flow{
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Request: &model.Request{
			Method:  "GET",
			URL:     "http://example.com",
			Host:    "example.com",
			Path:    "/",
			Proto:   "HTTP/1.1",
			Headers: http.Header{},
		},
		Response: &model.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Headers:    http.Header{},
		},
	}
	detail := vm.FormatFlowDetail(flow, false)
	if detail == "" {
		t.Error("should produce output for flow with response")
	}
}

// --- formatDuration ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{50, "50ms"},
		{999, "999ms"},
		{1000, "1.00s"},
		{1500, "1.50s"},
		{10000, "10.00s"},
	}

	for _, tt := range tests {
		d := time.Duration(tt.ms) * time.Millisecond
		got := formatDuration(d)
		if got != tt.want {
			t.Errorf("formatDuration(%dms) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

// --- getBaseDomain ---

func TestGetBaseDomain(t *testing.T) {
	tests := []struct {
		host, want string
	}{
		{"api.example.com", "example.com"},
		{"example.com", "example.com"},
		{"sub.api.example.com", "example.com"},
		{"example.com:443", "example.com"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		got := getBaseDomain(tt.host)
		if got != tt.want {
			t.Errorf("getBaseDomain(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

// --- SelectFlow ---

func TestSelectFlow_ValidIndex(t *testing.T) {
	vm, src := newTestVM(t)

	f := &model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com", Path: "/"},
	}
	src.flowStore.Add(f)
	vm.Refresh()

	vm.SelectFlow(0)
	if vm.GetSelectedFlow() == nil {
		t.Error("SelectFlow(0) should select a flow")
	}
	if vm.GetSelectedIndex() != 0 {
		t.Errorf("index = %d, want 0", vm.GetSelectedIndex())
	}
}

func TestSelectFlow_OutOfRange(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.SelectFlow(5) // no flows exist
	if vm.GetSelectedFlow() != nil {
		t.Error("out-of-range SelectFlow should set nil")
	}
	if vm.GetSelectedIndex() != -1 {
		t.Errorf("index = %d, want -1", vm.GetSelectedIndex())
	}
}

// --- SetSearchQuery ---

func TestSetSearchQuery(t *testing.T) {
	vm, src := newTestVM(t)

	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com/path", Host: "a.com", Path: "/path"},
	})
	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://b.com/path", Host: "b.com", Path: "/path"},
	})

	vm.SetSearchQuery("a.com")
	flows := vm.GetFilteredFlows()
	if len(flows) != 1 {
		t.Errorf("filtered flows = %d, want 1", len(flows))
	}
}

// --- SetFilterType ---

func TestSetFilterType(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.SetFilterType(model.FilterWhitelist)
	if vm.GetFilter().Type != model.FilterWhitelist {
		t.Error("SetFilterType should change the filter type")
	}
}

// --- ClearFlows ---

func TestClearFlows(t *testing.T) {
	vm, src := newTestVM(t)

	src.flowStore.Add(&model.Flow{
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com", Path: "/"},
	})
	vm.Refresh()

	vm.ClearFlows()
	if vm.GetFlowCount() != 0 {
		t.Error("ClearFlows should remove all flows")
	}
	if vm.GetSelectedFlow() != nil {
		t.Error("ClearFlows should clear selection")
	}
}

// --- Whitelist ops ---

func TestAddWhitelistPattern_Dedup(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.AddWhitelistPattern("a.com")
	vm.AddWhitelistPattern("a.com") // duplicate
	patterns := vm.GetWhitelistPatterns()
	if len(patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(patterns))
	}
}

func TestAddWhitelistPattern_Empty(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.AddWhitelistPattern("")
	if len(vm.GetWhitelistPatterns()) != 0 {
		t.Error("empty pattern should be rejected")
	}
}

func TestRemoveWhitelistPattern(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.AddWhitelistPattern("a.com")
	vm.RemoveWhitelistPattern("a.com")
	if len(vm.GetWhitelistPatterns()) != 0 {
		t.Error("pattern should be removed")
	}
}

func TestToggleWhitelistPattern(t *testing.T) {
	vm, src := newTestVM(t)

	vm.AddWhitelistPattern("a.com")
	if !src.sslProxyList.Match("a.com") {
		t.Error("pattern should be in SSL proxy list after add")
	}

	vm.ToggleWhitelistPattern("a.com")
	patterns := vm.GetWhitelistPatterns()
	if patterns[0].Enabled {
		t.Error("should be disabled after toggle")
	}
	if src.sslProxyList.Match("a.com") {
		t.Error("disabled pattern should be removed from SSL proxy list")
	}
}

func TestEditWhitelistPattern(t *testing.T) {
	vm, _ := newTestVM(t)

	vm.AddWhitelistPattern("old.com")
	vm.EditWhitelistPattern("old.com", "new.com")

	patterns := vm.GetWhitelistPatterns()
	if len(patterns) != 1 || patterns[0].Pattern != "new.com" {
		t.Errorf("after edit, patterns = %+v", patterns)
	}
}

// --- IsSecondary ---

func TestIsSecondary(t *testing.T) {
	vm, _ := newTestVM(t)

	if vm.IsSecondary() {
		t.Error("default should be false")
	}
	vm.SetSecondary(true)
	if !vm.IsSecondary() {
		t.Error("should be true after SetSecondary(true)")
	}
}
