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
	compiled    *regexp.Regexp // compiled pattern for regex matching
}

// NewMapRule creates a new mapping rule
func NewMapRule(ruleType MapRuleType, pattern, replacement string) *MapRule {
	rule := &MapRule{
		Type:        ruleType,
		Enabled:     true,
		Pattern:     pattern,
		Replacement: replacement,
	}
	rule.compile()
	return rule
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
	// Simple prefix/suffix matching
	if strings.HasSuffix(r.Pattern, "*") {
		return strings.HasPrefix(url, r.Pattern[:len(r.Pattern)-1])
	}
	if strings.HasPrefix(r.Pattern, "*") {
		return strings.HasSuffix(url, r.Pattern[1:])
	}
	return strings.Contains(url, r.Pattern)
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
func (s *MapRuleStore) FindMatch(url string) *MapRule {
	for _, r := range s.rules {
		if r.Match(url) {
			return r
		}
	}
	return nil
}

// All returns all rules
func (s *MapRuleStore) All() []*MapRule {
	return s.rules
}
