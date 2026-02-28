package model

import (
	"regexp"
	"strings"
	"sync"

	"nabu/internal/util"
)

// MapRuleType defines the type of mapping
type MapRuleType int

const (
	MapLocal MapRuleType = iota
	MapRemote
)

// Default priorities for rule types.
// Higher value = checked first by FindMatch.
const (
	PriorityMapLocal  = 100
	PriorityMapRemote = 50
)

func defaultPriority(t MapRuleType) int {
	switch t {
	case MapLocal:
		return PriorityMapLocal
	case MapRemote:
		return PriorityMapRemote
	default:
		return 0
	}
}

// MapRule defines a URL mapping rule
type MapRule struct {
	ID          int
	Type        MapRuleType
	Enabled     bool
	Name        string
	Method      string         // HTTP method filter (empty = match all)
	Pattern     string         // URL pattern to match
	Replacement string         // Local path or remote URL
	StatusCode  int            // HTTP status code (default 200)
	ContentType string         // Content-Type header (auto-detect if empty)
	Priority    int            // Higher priority rules are matched first
	compiled    *regexp.Regexp // compiled pattern for regex matching
}

// NewMapRule creates a new mapping rule
func NewMapRule(ruleType MapRuleType, pattern, replacement string) *MapRule {
	rule := &MapRule{
		Type:        ruleType,
		Enabled:     true,
		Pattern:     pattern,
		Replacement: replacement,
		StatusCode:  200,
		Priority:    defaultPriority(ruleType),
	}
	rule.compile()
	return rule
}

// NewMapLocalRule creates a new map local rule with all options
func NewMapLocalRule(pattern, localPath string, statusCode int, contentType string, method string) *MapRule {
	rule := &MapRule{
		Type:        MapLocal,
		Enabled:     true,
		Method:      strings.ToUpper(method),
		Pattern:     pattern,
		Replacement: localPath,
		StatusCode:  statusCode,
		ContentType: contentType,
		Priority:    PriorityMapLocal,
	}
	if rule.StatusCode == 0 {
		rule.StatusCode = 200
	}
	rule.compile()
	return rule
}

// NewMapRemoteRule creates a new map remote rule
func NewMapRemoteRule(pattern, remoteURL, method string) *MapRule {
	rule := &MapRule{
		Type:        MapRemote,
		Enabled:     true,
		Method:      strings.ToUpper(method),
		Pattern:     pattern,
		Replacement: remoteURL,
		Priority:    PriorityMapRemote,
	}
	rule.compile()
	return rule
}

// GetStatusCode returns the status code, defaulting to 200
func (r *MapRule) GetStatusCode() int {
	if r.StatusCode == 0 {
		return 200
	}
	return r.StatusCode
}

// GetContentType returns the content type, auto-detecting if not set
func (r *MapRule) GetContentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	return DetectContentType(r.Replacement)
}

// DetectContentType detects content type from file extension
func DetectContentType(path string) string {
	ext := strings.ToLower(getFileExt(path))

	switch ext {
	case ".json":
		return "application/json"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html", ".htm":
		return "text/html"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// getFileExt extracts file extension from path
func getFileExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

// compile prepares the pattern for matching
func (r *MapRule) compile() {
	// Try to compile as regex
	if strings.HasPrefix(r.Pattern, "^") || strings.HasPrefix(r.Pattern, "(") {
		re, err := regexp.Compile(r.Pattern)
		if err == nil {
			r.compiled = re
		}
	}
}

// MatchRequest checks if a URL and method match this rule
func (r *MapRule) MatchRequest(url, method string) bool {
	if !r.Enabled {
		return false
	}
	// Check method filter (empty = match all)
	if r.Method != "" && !strings.EqualFold(r.Method, method) {
		return false
	}
	if r.compiled != nil {
		return r.compiled.MatchString(url)
	}

	url = strings.ToLower(url)
	pattern := strings.ToLower(r.Pattern)

	// Glob pattern with *
	if strings.Contains(pattern, "*") {
		return util.MatchGlob(url, pattern)
	}

	// Exact match (with optional trailing slash)
	if url == pattern {
		return true
	}
	// Match pattern without trailing slash to URL with trailing slash
	if url == pattern+"/" {
		return true
	}
	// Match pattern with trailing slash to URL without trailing slash
	if strings.HasSuffix(pattern, "/") && url == strings.TrimSuffix(pattern, "/") {
		return true
	}

	return false
}

// Apply returns the replacement URL for a matched URL
func (r *MapRule) Apply(url string) string {
	if r.compiled != nil {
		return r.compiled.ReplaceAllString(url, r.Replacement)
	}
	// Simple replacement
	if strings.HasSuffix(r.Pattern, "*") {
		prefix := r.Pattern[:len(r.Pattern)-1]
		return r.Replacement + strings.TrimPrefix(url, prefix)
	}
	return r.Replacement
}

// MapRuleStore manages mapping rules
type MapRuleStore struct {
	mu     sync.RWMutex
	rules  []*MapRule
	nextID int
}

// NewMapRuleStore creates a new rule store
func NewMapRuleStore() *MapRuleStore {
	return &MapRuleStore{
		rules:  make([]*MapRule, 0),
		nextID: 1,
	}
}

// Add adds a new rule
func (s *MapRuleStore) Add(rule *MapRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule.ID = s.nextID
	s.nextID++
	s.rules = append(s.rules, rule)
}

// Remove removes a rule by ID
func (s *MapRuleStore) Remove(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.rules {
		if r.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return
		}
	}
}

// Toggle toggles a rule's enabled state
func (s *MapRuleStore) Toggle(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.rules {
		if r.ID == id {
			r.Enabled = !r.Enabled
			return
		}
	}
}

// FindMatch finds the highest-priority matching rule for a URL and method.
// Rules with higher Priority values are preferred. Among rules with equal
// priority, the first one added wins (stable, insertion-order tiebreak).
func (s *MapRuleStore) FindMatch(url, method string) *MapRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *MapRule
	for _, r := range s.rules {
		if r.MatchRequest(url, method) && (best == nil || r.Priority > best.Priority) {
			best = r
		}
	}
	return best
}

// All returns all rules
func (s *MapRuleStore) All() []*MapRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*MapRule, len(s.rules))
	copy(result, s.rules)
	return result
}

// Clear removes all rules
func (s *MapRuleStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = make([]*MapRule, 0)
}

// GetByID returns a rule by ID
func (s *MapRuleStore) GetByID(id int) *MapRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.rules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Update updates an existing rule
func (s *MapRuleStore) Update(rule *MapRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.rules {
		if r.ID == rule.ID {
			s.rules[i] = rule
			return
		}
	}
}
