package model

import (
	"sync"
	"testing"
)

// --- DetectContentType ---

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"file.json", "application/json"},
		{"file.js", "application/javascript"},
		{"file.css", "text/css"},
		{"file.html", "text/html"},
		{"file.htm", "text/html"},
		{"file.xml", "application/xml"},
		{"file.txt", "text/plain"},
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.jpeg", "image/jpeg"},
		{"file.gif", "image/gif"},
		{"file.svg", "image/svg+xml"},
		{"file.ico", "image/x-icon"},
		{"file.woff", "font/woff"},
		{"file.woff2", "font/woff2"},
		{"file.ttf", "font/ttf"},
		{"file.pdf", "application/pdf"},
		{"file.unknown", "application/octet-stream"},
		{"noext", "application/octet-stream"},
		{"FILE.JSON", "application/json"}, // case-insensitive
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectContentType(tt.path)
			if got != tt.want {
				t.Errorf("DetectContentType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// --- MapRule.Match ---

func TestMapRule_Match(t *testing.T) {
	tests := []struct {
		name    string
		rule    *MapRule
		url     string
		want    bool
	}{
		{"exact match", NewMapRule(MapLocal, "http://example.com/api", "/tmp/f"), "http://example.com/api", true},
		{"trailing slash on URL", NewMapRule(MapLocal, "http://example.com/api", "/tmp/f"), "http://example.com/api/", true},
		{"trailing slash on pattern", NewMapRule(MapLocal, "http://example.com/api/", "/tmp/f"), "http://example.com/api", true},
		{"no match", NewMapRule(MapLocal, "http://example.com/api", "/tmp/f"), "http://other.com/api", false},
		{"glob star", NewMapRule(MapLocal, "http://example.com/*", "/tmp/f"), "http://example.com/anything", true},
		{"glob star with question", NewMapRule(MapLocal, "http://example.com/ab?/*", "/tmp/f"), "http://example.com/abc/d", true},
		{"glob star with question no match", NewMapRule(MapLocal, "http://example.com/ab?/*", "/tmp/f"), "http://example.com/abcd/e", false},
		{"regex", NewMapRule(MapLocal, "^http://example\\.com/api/v[0-9]+", "/tmp/f"), "http://example.com/api/v2", true},
		{"regex no match", NewMapRule(MapLocal, "^http://example\\.com/api/v[0-9]+", "/tmp/f"), "http://example.com/api/vx", false},
		{"case insensitive glob", NewMapRule(MapLocal, "HTTP://EXAMPLE.COM/*", "/tmp/f"), "http://example.com/path", true},
		{"disabled rule", func() *MapRule {
			r := NewMapRule(MapLocal, "http://example.com/*", "/tmp/f")
			r.Enabled = false
			return r
		}(), "http://example.com/path", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Match(tt.url)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// --- MapRule.Apply ---

func TestMapRule_Apply(t *testing.T) {
	tests := []struct {
		name        string
		rule        *MapRule
		url         string
		want        string
	}{
		{
			"simple replacement",
			NewMapRule(MapLocal, "http://example.com/api", "/tmp/response.json"),
			"http://example.com/api",
			"/tmp/response.json",
		},
		{
			"glob prefix replacement",
			NewMapRule(MapLocal, "http://example.com/*", "/local/"),
			"http://example.com/foo/bar",
			"/local/foo/bar",
		},
		{
			"regex with capture groups",
			NewMapRule(MapLocal, "^http://example\\.com/api/(v[0-9]+)/(.+)", "/local/$1/$2"),
			"http://example.com/api/v2/users",
			"/local/v2/users",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Apply(tt.url)
			if got != tt.want {
				t.Errorf("Apply(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

// --- MapRuleStore CRUD ---

func TestMapRuleStore_Add(t *testing.T) {
	store := NewMapRuleStore()
	r1 := NewMapRule(MapLocal, "p1", "r1")
	r2 := NewMapRule(MapLocal, "p2", "r2")
	store.Add(r1)
	store.Add(r2)

	if r1.ID != 1 {
		t.Errorf("first rule ID = %d, want 1", r1.ID)
	}
	if r2.ID != 2 {
		t.Errorf("second rule ID = %d, want 2", r2.ID)
	}
	if len(store.All()) != 2 {
		t.Errorf("All() length = %d, want 2", len(store.All()))
	}
}

func TestMapRuleStore_Remove(t *testing.T) {
	store := NewMapRuleStore()
	r := NewMapRule(MapLocal, "p", "r")
	store.Add(r)
	store.Remove(r.ID)
	if len(store.All()) != 0 {
		t.Errorf("All() length after Remove = %d, want 0", len(store.All()))
	}
}

func TestMapRuleStore_Toggle(t *testing.T) {
	store := NewMapRuleStore()
	r := NewMapRule(MapLocal, "p", "r")
	store.Add(r)

	if !r.Enabled {
		t.Fatal("new rule should be enabled")
	}
	store.Toggle(r.ID)
	got := store.GetByID(r.ID)
	if got.Enabled {
		t.Error("after Toggle, rule should be disabled")
	}
}

func TestMapRuleStore_GetByID(t *testing.T) {
	store := NewMapRuleStore()
	r := NewMapRule(MapLocal, "p", "r")
	store.Add(r)

	got := store.GetByID(r.ID)
	if got == nil || got.Pattern != "p" {
		t.Error("GetByID should return the added rule")
	}
	if store.GetByID(999) != nil {
		t.Error("GetByID should return nil for unknown ID")
	}
}

func TestMapRuleStore_Update(t *testing.T) {
	store := NewMapRuleStore()
	r := NewMapRule(MapLocal, "p", "r")
	store.Add(r)

	updated := NewMapRule(MapLocal, "p2", "r2")
	updated.ID = r.ID
	store.Update(updated)

	got := store.GetByID(r.ID)
	if got.Pattern != "p2" {
		t.Errorf("after Update, pattern = %q, want %q", got.Pattern, "p2")
	}
}

func TestMapRuleStore_Clear(t *testing.T) {
	store := NewMapRuleStore()
	store.Add(NewMapRule(MapLocal, "p1", "r1"))
	store.Add(NewMapRule(MapLocal, "p2", "r2"))
	store.Clear()

	if len(store.All()) != 0 {
		t.Errorf("All() after Clear = %d, want 0", len(store.All()))
	}
	// nextID should NOT reset — add another and check
	r := NewMapRule(MapLocal, "p3", "r3")
	store.Add(r)
	if r.ID <= 2 {
		t.Errorf("after Clear, nextID should keep incrementing; got ID %d", r.ID)
	}
}

// --- MapRuleStore.FindMatch ---

func TestMapRuleStore_FindMatch(t *testing.T) {
	store := NewMapRuleStore()

	// Add a remote rule first, then a local rule
	remote := NewMapRule(MapRemote, "http://example.com/*", "http://other.com")
	local := NewMapRule(MapLocal, "http://example.com/*", "/tmp/local")
	store.Add(remote)
	store.Add(local)

	// MapLocal should win even though it was added second
	match := store.FindMatch("http://example.com/path")
	if match == nil {
		t.Fatal("FindMatch returned nil")
	}
	if match.Type != MapLocal {
		t.Error("FindMatch should prioritize MapLocal over MapRemote")
	}

	// No match
	if store.FindMatch("http://other.example.com/path") != nil {
		t.Error("FindMatch should return nil for non-matching URL")
	}
}

// --- Concurrency ---

func TestMapRuleStore_Concurrent(t *testing.T) {
	store := NewMapRuleStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			store.Add(NewMapRule(MapLocal, "pattern", "replacement"))
		}(i)
		go func() {
			defer wg.Done()
			store.All()
		}()
		go func() {
			defer wg.Done()
			store.FindMatch("http://example.com/path")
		}()
	}
	wg.Wait()
}

// --- Constructors ---

func TestNewMapRule(t *testing.T) {
	r := NewMapRule(MapLocal, "p", "r")
	if !r.Enabled {
		t.Error("NewMapRule should be enabled by default")
	}
	if r.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", r.StatusCode)
	}
}

func TestNewMapLocalRule(t *testing.T) {
	r := NewMapLocalRule("p", "/tmp/f", 0, "")
	if r.Type != MapLocal {
		t.Error("type should be MapLocal")
	}
	if r.StatusCode != 200 {
		t.Error("StatusCode should default to 200 when 0 is passed")
	}
	if !r.Enabled {
		t.Error("should be enabled by default")
	}
}

func TestNewMapRemoteRule(t *testing.T) {
	r := NewMapRemoteRule("p", "http://other.com")
	if r.Type != MapRemote {
		t.Error("type should be MapRemote")
	}
	if !r.Enabled {
		t.Error("should be enabled by default")
	}
}

// --- GetStatusCode / GetContentType defaults ---

func TestGetStatusCode(t *testing.T) {
	r := &MapRule{StatusCode: 0}
	if r.GetStatusCode() != 200 {
		t.Errorf("GetStatusCode() = %d, want 200 default", r.GetStatusCode())
	}
	r.StatusCode = 404
	if r.GetStatusCode() != 404 {
		t.Errorf("GetStatusCode() = %d, want 404", r.GetStatusCode())
	}
}

func TestGetContentType(t *testing.T) {
	r := &MapRule{ContentType: "text/html", Replacement: "/tmp/file.json"}
	if r.GetContentType() != "text/html" {
		t.Error("should return explicit ContentType when set")
	}
	r.ContentType = ""
	if r.GetContentType() != "application/json" {
		t.Error("should auto-detect from Replacement when ContentType is empty")
	}
}
