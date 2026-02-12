package viewmodel

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"proxy-tui/internal/model"
)

// --- FormatCURL ---

func TestFormatCURL_GET(t *testing.T) {
	flow := &model.Flow{
		Request: &model.Request{
			Method:  "GET",
			URL:     "http://example.com/api",
			Host:    "example.com",
			Headers: http.Header{"Accept": []string{"application/json"}},
		},
	}

	curl, err := FormatCURL(flow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(curl, "curl") {
		t.Errorf("should start with curl: %q", curl)
	}
	if strings.Contains(curl, "-X") {
		t.Error("GET should not include -X flag")
	}
	if !strings.Contains(curl, "example.com/api") {
		t.Error("should contain the URL")
	}
	if !strings.Contains(curl, "Accept") {
		t.Error("should contain the Accept header")
	}
}

func TestFormatCURL_POST(t *testing.T) {
	flow := &model.Flow{
		Request: &model.Request{
			Method:  "POST",
			URL:     "http://example.com/api",
			Host:    "example.com",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"key":"value"}`),
		},
	}

	curl, err := FormatCURL(flow)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(curl, "-X POST") {
		t.Error("POST should include -X POST")
	}
	if !strings.Contains(curl, "-d") {
		t.Error("should include -d for body")
	}
}

func TestFormatCURL_NilFlow(t *testing.T) {
	_, err := FormatCURL(nil)
	if err == nil {
		t.Error("should error on nil flow")
	}
}

func TestFormatCURL_Tunneled(t *testing.T) {
	flow := &model.Flow{
		Tunneled: true,
		Request:  &model.Request{Method: "CONNECT", Host: "example.com"},
	}
	_, err := FormatCURL(flow)
	if err == nil {
		t.Error("should error on tunneled flow")
	}
}

// --- FormatHAR ---

func TestFormatHAR_SingleFlow(t *testing.T) {
	now := time.Now()
	flow := &model.Flow{
		StartTime: now,
		EndTime:   now.Add(100 * time.Millisecond),
		Request: &model.Request{
			Method:  "GET",
			URL:     "http://example.com/api",
			Host:    "example.com",
			Proto:   "HTTP/1.1",
			Headers: http.Header{},
		},
		Response: &model.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"result":true}`),
		},
	}

	data, err := FormatHAR([]*model.Flow{flow})
	if err != nil {
		t.Fatal(err)
	}

	var har HARLog
	if err := json.Unmarshal(data, &har); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if har.Log.Version != "1.2" {
		t.Errorf("version = %q, want 1.2", har.Log.Version)
	}
	if len(har.Log.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(har.Log.Entries))
	}
	entry := har.Log.Entries[0]
	if entry.Request.Method != "GET" {
		t.Errorf("method = %q", entry.Request.Method)
	}
	if entry.Response.Status != 200 {
		t.Errorf("status = %d", entry.Response.Status)
	}
	if entry.Response.StatusText != "OK" {
		t.Errorf("statusText = %q, want OK", entry.Response.StatusText)
	}
}

func TestFormatHAR_SkipsTunneled(t *testing.T) {
	flows := []*model.Flow{
		{Tunneled: true, Request: &model.Request{Method: "CONNECT"}},
		{Request: &model.Request{Method: "GET", URL: "http://a.com", Headers: http.Header{}},
			Response: &model.Response{StatusCode: 200, Headers: http.Header{}},
			StartTime: time.Now(), EndTime: time.Now()},
	}

	data, err := FormatHAR(flows)
	if err != nil {
		t.Fatal(err)
	}

	var har HARLog
	json.Unmarshal(data, &har)
	if len(har.Log.Entries) != 1 {
		t.Errorf("should skip tunneled, got %d entries", len(har.Log.Entries))
	}
}

func TestFormatHAR_Empty(t *testing.T) {
	data, err := FormatHAR(nil)
	if err != nil {
		t.Fatal(err)
	}

	var har HARLog
	json.Unmarshal(data, &har)
	if len(har.Log.Entries) != 0 {
		t.Error("empty input should produce 0 entries")
	}
}

// --- shellQuote ---

// --- ParseHAR ---

func TestParseHAR_RoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	flows := []*model.Flow{
		{
			StartTime: now,
			EndTime:   now.Add(150 * time.Millisecond),
			Request: &model.Request{
				Method:  "POST",
				URL:     "https://api.example.com/users?page=2",
				Host:    "api.example.com",
				Path:    "/users?page=2",
				Proto:   "HTTP/1.1",
				Headers: http.Header{"Content-Type": []string{"application/json"}},
				Body:    []byte(`{"name":"test"}`),
			},
			Response: &model.Response{
				StatusCode: 201,
				Status:     "201 Created",
				Proto:      "HTTP/1.1",
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"id":42}`),
			},
		},
		{
			StartTime: now.Add(time.Second),
			EndTime:   now.Add(time.Second + 50*time.Millisecond),
			Request: &model.Request{
				Method:  "GET",
				URL:     "https://api.example.com/health",
				Host:    "api.example.com",
				Path:    "/health",
				Proto:   "HTTP/1.1",
				Headers: http.Header{},
			},
			Response: &model.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Proto:      "HTTP/1.1",
				Headers:    http.Header{},
				Body:       []byte("ok"),
			},
		},
	}

	// Export to HAR
	data, err := FormatHAR(flows)
	if err != nil {
		t.Fatalf("FormatHAR: %v", err)
	}

	// Import back
	parsed, err := ParseHAR(data)
	if err != nil {
		t.Fatalf("ParseHAR: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("got %d flows, want 2", len(parsed))
	}

	// Verify first flow
	f := parsed[0]
	if f.Request.Method != "POST" {
		t.Errorf("flow[0] method = %q, want POST", f.Request.Method)
	}
	if f.Request.URL != "https://api.example.com/users?page=2" {
		t.Errorf("flow[0] URL = %q", f.Request.URL)
	}
	if f.Request.Host != "api.example.com" {
		t.Errorf("flow[0] host = %q", f.Request.Host)
	}
	if f.Request.Path != "/users?page=2" {
		t.Errorf("flow[0] path = %q", f.Request.Path)
	}
	if string(f.Request.Body) != `{"name":"test"}` {
		t.Errorf("flow[0] request body = %q", f.Request.Body)
	}
	if f.Response == nil {
		t.Fatal("flow[0] response is nil")
	}
	if f.Response.StatusCode != 201 {
		t.Errorf("flow[0] status = %d, want 201", f.Response.StatusCode)
	}
	if string(f.Response.Body) != `{"id":42}` {
		t.Errorf("flow[0] response body = %q", f.Response.Body)
	}

	// Verify second flow
	f2 := parsed[1]
	if f2.Request.Method != "GET" {
		t.Errorf("flow[1] method = %q, want GET", f2.Request.Method)
	}
	if f2.Response.StatusCode != 200 {
		t.Errorf("flow[1] status = %d, want 200", f2.Response.StatusCode)
	}
}

func TestParseHAR_NoResponse(t *testing.T) {
	// Flow with no response should still round-trip (response will be nil after parse)
	now := time.Now()
	flows := []*model.Flow{
		{
			StartTime: now,
			EndTime:   now,
			Request: &model.Request{
				Method:  "GET",
				URL:     "http://example.com/pending",
				Host:    "example.com",
				Path:    "/pending",
				Proto:   "HTTP/1.1",
				Headers: http.Header{},
			},
		},
	}

	data, err := FormatHAR(flows)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseHAR(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed) != 1 {
		t.Fatalf("got %d flows, want 1", len(parsed))
	}
	if parsed[0].Response != nil {
		t.Error("incomplete flow should have nil response after parse")
	}
}

func TestParseHAR_InvalidJSON(t *testing.T) {
	_, err := ParseHAR([]byte("not json"))
	if err == nil {
		t.Error("should error on invalid JSON")
	}
}

func TestParseHAR_EmptyEntries(t *testing.T) {
	data := []byte(`{"log":{"version":"1.2","creator":{"name":"test","version":"1.0"},"entries":[]}}`)
	flows, err := ParseHAR(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(flows) != 0 {
		t.Errorf("got %d flows, want 0", len(flows))
	}
}

// --- shellQuote ---

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"simple", "'simple'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.in)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
