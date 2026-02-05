package model

import (
	"regexp"
	"strings"
)

// MapRuleType defines the type of mapping
type MapRuleType int

const (
	MapLocal MapRuleType = iota
	MapRemote
)

// MapRule defines a URL mapping rule
type MapRule struct {
	ID          int
	Type        MapRuleType
	Enabled     bool
	Name        string
	Pattern     string         // URL pattern to match
	Replacement string         // Local path or remote URL
	StatusCode  int            // HTTP status code (default 200)
	ContentType string         // Content-Type header (auto-detect if empty)
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
	}
	rule.compile()
	return rule
}

// NewMapLocalRule creates a new map local rule with all options
func NewMapLocalRule(pattern, localPath string, statusCode int, contentType string) *MapRule {
	rule := &MapRule{
		Type:        MapLocal,
		Enabled:     true,
		Pattern:     pattern,
		Replacement: localPath,
		StatusCode:  statusCode,
		ContentType: contentType,
	}
	if rule.StatusCode == 0 {
		rule.StatusCode = 200
	}
	rule.compile()
	return rule
}

// NewMapRemoteRule creates a new map remote rule
func NewMapRemoteRule(pattern, remoteURL string) *MapRule {
	rule := &MapRule{
		Type:        MapRemote,
		Enabled:     true,
		Pattern:     pattern,
		Replacement: remoteURL,
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

// Match checks if a URL matches this rule
func (r *MapRule) Match(url string) bool {
	if !r.Enabled {
		return false
	}
	if r.compiled != nil {
		return r.compiled.MatchString(url)
	}

	url = strings.ToLower(url)
	pattern := strings.ToLower(r.Pattern)

	// Glob pattern with *
	if strings.Contains(pattern, "*") {
		return matchGlobPattern(url, pattern)
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

// matchGlobPattern matches a URL against a glob pattern
func matchGlobPattern(url, pattern string) bool {
	// Convert glob to regex
	regexPattern := "^"
	for _, c := range pattern {
		switch c {
		case '*':
			regexPattern += ".*"
		case '?':
			regexPattern += "."
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			regexPattern += "\\" + string(c)
		default:
			regexPattern += string(c)
		}
	}
	regexPattern += "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}
	return re.MatchString(url)
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
	rule.ID = s.nextID
	s.nextID++
	s.rules = append(s.rules, rule)
}

// Remove removes a rule by ID
func (s *MapRuleStore) Remove(id int) {
	for i, r := range s.rules {
		if r.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return
		}
	}
}

// Toggle toggles a rule's enabled state
func (s *MapRuleStore) Toggle(id int) {
	for _, r := range s.rules {
		if r.ID == id {
			r.Enabled = !r.Enabled
			return
		}
	}
}

// FindMatch finds the first matching rule for a URL
// Map Local rules are always checked first, then Map Remote rules
func (s *MapRuleStore) FindMatch(url string) *MapRule {
	// First pass: check Map Local rules
	for _, r := range s.rules {
		if r.Type == MapLocal && r.Match(url) {
			return r
		}
	}
	// Second pass: check Map Remote rules
	for _, r := range s.rules {
		if r.Type == MapRemote && r.Match(url) {
			return r
		}
	}
	return nil
}

// All returns all rules
func (s *MapRuleStore) All() []*MapRule {
	return s.rules
}

// Clear removes all rules
func (s *MapRuleStore) Clear() {
	s.rules = make([]*MapRule, 0)
}

// GetByID returns a rule by ID
func (s *MapRuleStore) GetByID(id int) *MapRule {
	for _, r := range s.rules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Update updates an existing rule
func (s *MapRuleStore) Update(rule *MapRule) {
	for i, r := range s.rules {
		if r.ID == rule.ID {
			s.rules[i] = rule
			return
		}
	}
}
