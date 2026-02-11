package model

import (
	"net/http"
	"testing"
)

// --- FilterState.Match ---

func TestFilterState_Match_NilFlow(t *testing.T) {
	f := NewFilterState()
	if f.Match(nil) {
		t.Error("nil flow should not match")
	}
}

func TestFilterState_Match_NilRequest(t *testing.T) {
	f := NewFilterState()
	if f.Match(&Flow{}) {
		t.Error("flow with nil Request should not match")
	}
}

func TestFilterState_Match_EmptyFilter(t *testing.T) {
	f := NewFilterState()
	flow := &Flow{Request: &Request{URL: "http://example.com", Host: "example.com", Method: "GET"}}
	if !f.Match(flow) {
		t.Error("empty filter should match everything")
	}
}

// --- SearchQuery ---

func TestFilterState_SearchQuery(t *testing.T) {
	f := NewFilterState()
	f.SearchQuery = "example"

	flow := &Flow{Request: &Request{URL: "http://example.com/path", Host: "example.com", Method: "GET"}}
	if !f.Match(flow) {
		t.Error("should match URL containing query")
	}

	flow2 := &Flow{Request: &Request{URL: "http://other.com/path", Host: "other.com", Method: "GET"}}
	if f.Match(flow2) {
		t.Error("should not match when query not in URL or Host")
	}

	// Case insensitive
	f.SearchQuery = "EXAMPLE"
	if !f.Match(flow) {
		t.Error("search should be case-insensitive")
	}
}

// --- HostPatterns ---

func TestFilterState_HostPatterns_Whitelist(t *testing.T) {
	f := NewFilterState()
	f.Type = FilterWhitelist
	f.HostPatterns = []HostPattern{
		{Pattern: "*.example.com", Enabled: true},
	}

	flow := &Flow{Request: &Request{URL: "http://api.example.com/path", Host: "api.example.com", Method: "GET"}}
	if !f.Match(flow) {
		t.Error("should match host pattern")
	}

	flow2 := &Flow{Request: &Request{URL: "http://other.com/path", Host: "other.com", Method: "GET"}}
	if f.Match(flow2) {
		t.Error("should not match non-whitelisted host")
	}
}

func TestFilterState_HostPatterns_DisabledSkipped(t *testing.T) {
	f := NewFilterState()
	f.Type = FilterWhitelist
	f.HostPatterns = []HostPattern{
		{Pattern: "*.example.com", Enabled: false},
	}

	flow := &Flow{Request: &Request{URL: "http://other.com/path", Host: "other.com", Method: "GET"}}
	// All patterns disabled → hasEnabled=false → no host filtering
	if !f.Match(flow) {
		t.Error("all patterns disabled should pass through")
	}
}

func TestFilterState_HostPatterns_Regex(t *testing.T) {
	f := NewFilterState()
	f.Type = FilterWhitelist
	f.HostPatterns = []HostPattern{
		{Pattern: "^api\\.", Enabled: true},
	}

	flow := &Flow{Request: &Request{URL: "http://api.example.com/path", Host: "api.example.com", Method: "GET"}}
	if !f.Match(flow) {
		t.Error("regex pattern should match")
	}
}

// --- Methods filter ---

func TestFilterState_Methods(t *testing.T) {
	f := NewFilterState()
	f.Methods = []string{"GET", "POST"}

	flow := &Flow{Request: &Request{URL: "http://example.com", Host: "example.com", Method: "GET"}}
	if !f.Match(flow) {
		t.Error("GET should match")
	}

	flow2 := &Flow{Request: &Request{URL: "http://example.com", Host: "example.com", Method: "DELETE"}}
	if f.Match(flow2) {
		t.Error("DELETE should not match")
	}
}

// --- StatusCodes filter ---

func TestFilterState_StatusCodes(t *testing.T) {
	f := NewFilterState()
	f.StatusCodes = []int{200, 404}

	flow := &Flow{
		Request:  &Request{URL: "http://example.com", Host: "example.com", Method: "GET"},
		Response: &Response{StatusCode: 200},
	}
	if !f.Match(flow) {
		t.Error("200 should match")
	}

	flow2 := &Flow{
		Request:  &Request{URL: "http://example.com", Host: "example.com", Method: "GET"},
		Response: &Response{StatusCode: 500},
	}
	if f.Match(flow2) {
		t.Error("500 should not match")
	}
}

func TestFilterState_StatusCodes_NoResponse(t *testing.T) {
	f := NewFilterState()
	f.StatusCodes = []int{200}

	flow := &Flow{
		Request: &Request{URL: "http://example.com", Host: "example.com", Method: "GET"},
	}
	// No Response → status code filter is skipped
	if !f.Match(flow) {
		t.Error("flow without response should pass status code filter")
	}
}

// --- Combined ---

func TestFilterState_Combined(t *testing.T) {
	f := NewFilterState()
	f.SearchQuery = "example"
	f.Methods = []string{"GET"}
	f.StatusCodes = []int{200}
	f.Type = FilterWhitelist
	f.HostPatterns = []HostPattern{
		{Pattern: "*.example.com", Enabled: true},
	}

	flow := &Flow{
		Request: &Request{
			URL:    "http://api.example.com/path",
			Host:   "api.example.com",
			Method: "GET",
		},
		Response: &Response{StatusCode: 200},
	}
	if !f.Match(flow) {
		t.Error("flow matching all criteria should match")
	}

	// Wrong method
	flow2 := &Flow{
		Request: &Request{
			URL:    "http://api.example.com/path",
			Host:   "api.example.com",
			Method: "POST",
		},
		Response: &Response{StatusCode: 200},
	}
	if f.Match(flow2) {
		t.Error("wrong method should not match")
	}
}

// --- matchPattern ---

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		pattern string
		want    bool
	}{
		{"exact", "example.com", "example.com", true},
		{"wildcard star", "anything.com", "*", true},
		{"wildcard subdomain", "api.example.com", "*.example.com", true},
		{"wildcard subdomain bare domain", "example.com", "*.example.com", true},
		{"general glob", "api.google.com", "*google*", true},
		{"regex", "api.example.com", "^api\\.", true},
		{"case insensitive", "Example.COM", "example.com", true},
		{"no match", "other.com", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.host, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

// --- FilterState with actual headers ---

func TestFilterState_Match_FlowWithHeaders(t *testing.T) {
	f := NewFilterState()
	flow := &Flow{
		Request: &Request{
			URL:     "http://example.com/path",
			Host:    "example.com",
			Method:  "GET",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
		},
	}
	if !f.Match(flow) {
		t.Error("should match flow with headers")
	}
}
