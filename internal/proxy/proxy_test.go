package proxy

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// --- SSLProxyList ---

func TestSSLProxyList_Add_Dedup(t *testing.T) {
	l := NewSSLProxyList()
	l.Add("example.com")
	l.Add("example.com") // duplicate
	l.Add("other.com")

	if len(l.Patterns()) != 2 {
		t.Errorf("Patterns count = %d, want 2", len(l.Patterns()))
	}
}

func TestSSLProxyList_Remove(t *testing.T) {
	l := NewSSLProxyList()
	l.Add("a.com")
	l.Add("b.com")
	l.Remove("a.com")

	patterns := l.Patterns()
	if len(patterns) != 1 || patterns[0] != "b.com" {
		t.Errorf("after Remove, patterns = %v, want [b.com]", patterns)
	}
}

func TestSSLProxyList_Clear(t *testing.T) {
	l := NewSSLProxyList()
	l.Add("a.com")
	l.Clear()
	if len(l.Patterns()) != 0 {
		t.Error("Clear should remove all patterns")
	}
}

func TestSSLProxyList_Patterns_ReturnsCopy(t *testing.T) {
	l := NewSSLProxyList()
	l.Add("a.com")
	p := l.Patterns()
	p[0] = "mutated"
	if l.Patterns()[0] != "a.com" {
		t.Error("mutating Patterns() result should not affect list")
	}
}

func TestSSLProxyList_Match(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		host     string
		want     bool
	}{
		{"empty list", nil, "anything.com", false},
		{"exact", []string{"example.com"}, "example.com", true},
		{"port stripping", []string{"example.com"}, "example.com:443", true},
		{"wildcard subdomain", []string{"*.example.com"}, "api.example.com", true},
		{"wildcard subdomain bare", []string{"*.example.com"}, "example.com", true},
		{"star all", []string{"*"}, "anything.com", true},
		{"glob pattern", []string{"*google*"}, "maps.google.com", true},
		{"regex", []string{"^api\\."}, "api.example.com", true},
		{"no match", []string{"example.com"}, "other.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewSSLProxyList()
			for _, p := range tt.patterns {
				l.Add(p)
			}
			got := l.Match(tt.host)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestSSLProxyList_Concurrent(t *testing.T) {
	l := NewSSLProxyList()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			l.Add("pattern.com")
		}(i)
		go func() {
			defer wg.Done()
			l.Match("pattern.com")
		}()
		go func() {
			defer wg.Done()
			l.Remove("pattern.com")
		}()
	}
	wg.Wait()
}

// --- stripJSONComments ---

func TestStripJSONComments(t *testing.T) {
	input := `{
  // this is a comment
  "key": "value", // inline comment
  "url": "http://example.com"
}
`
	got := string(stripJSONComments([]byte(input)))
	if strings.Contains(got, "// this is a comment") {
		t.Error("line comment should be stripped")
	}
	if strings.Contains(got, "// inline comment") {
		t.Error("inline comment should be stripped")
	}
	if !strings.Contains(got, `"key": "value"`) {
		t.Error("values should be preserved")
	}
}

func TestStripJSONComments_InsideString(t *testing.T) {
	input := `{"url": "http://example.com//path"}`
	got := string(stripJSONComments([]byte(input)))
	if !strings.Contains(got, "http://example.com//path") {
		t.Error("// inside string should be preserved")
	}
}

// --- parseJSONCResponseFile ---

func TestParseJSONCResponseFile_Valid(t *testing.T) {
	content := []byte(`{
  "status": 201,
  "headers": {"Content-Type": "application/json"},
  "body": {"id": 1}
}`)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseJSONCResponseFile(content, req, "/tmp/test.jsonc")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
}

func TestParseJSONCResponseFile_DefaultStatus(t *testing.T) {
	content := []byte(`{"body": "hello"}`)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseJSONCResponseFile(content, req, "/tmp/test.jsonc")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200 (default)", resp.StatusCode)
	}
}

func TestParseJSONCResponseFile_InvalidJSON(t *testing.T) {
	content := []byte(`not json`)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseJSONCResponseFile(content, req, "/tmp/test.jsonc")
	if resp != nil {
		t.Error("invalid JSON should return nil")
	}
}

// --- buildResponseFromJSON ---

func TestBuildResponseFromJSON_BodyTypes(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// String body
	resp := buildResponseFromJSON(&MapLocalResponse{Status: 200, Body: "hello"}, req, "/tmp/f")
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("string body = %q, want %q", body, "hello")
	}

	// Object body
	resp = buildResponseFromJSON(&MapLocalResponse{Status: 200, Body: map[string]interface{}{"k": "v"}}, req, "/tmp/f")
	body, _ = io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"k"`) {
		t.Error("object body should be marshaled as JSON")
	}

	// Array body
	resp = buildResponseFromJSON(&MapLocalResponse{Status: 200, Body: []interface{}{"a", "b"}}, req, "/tmp/f")
	body, _ = io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"a"`) {
		t.Error("array body should be marshaled as JSON")
	}

	// Nil body
	resp = buildResponseFromJSON(&MapLocalResponse{Status: 200, Body: nil}, req, "/tmp/f")
	body, _ = io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Error("nil body should produce empty body")
	}
}

// --- parseOldHTTPFormat ---

func TestParseOldHTTPFormat_Valid(t *testing.T) {
	content := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World")
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseOldHTTPFormat(content, req, "/tmp/f")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello World" {
		t.Errorf("body = %q, want %q", body, "Hello World")
	}
}

func TestParseOldHTTPFormat_NoSeparator(t *testing.T) {
	content := []byte("just some text without separator")
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseOldHTTPFormat(content, req, "/tmp/f")
	if resp != nil {
		t.Error("no separator should return nil")
	}
}

func TestParseOldHTTPFormat_NoHTTPPrefix(t *testing.T) {
	content := []byte("Not HTTP\n\nbody")
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := parseOldHTTPFormat(content, req, "/tmp/f")
	if resp != nil {
		t.Error("no HTTP/ prefix should return nil")
	}
}

// --- DefaultConfig ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want 0.0.0.0", cfg.BindAddress)
	}
	if cfg.MaxFlows != 10000 {
		t.Errorf("MaxFlows = %d, want 10000", cfg.MaxFlows)
	}
	if cfg.Verbose {
		t.Error("Verbose should default to false")
	}
	if cfg.EventChanSize != 1000 {
		t.Errorf("EventChanSize = %d, want 1000", cfg.EventChanSize)
	}
}
