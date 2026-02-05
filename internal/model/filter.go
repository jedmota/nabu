package model

import (
	"regexp"
	"strings"
)

// FilterType represents different filter modes
type FilterType int

const (
	FilterAll FilterType = iota
	FilterWhitelist
	FilterCustom
)

// HostPattern represents a whitelist pattern with enabled state
type HostPattern struct {
	Pattern string
	Enabled bool
}

// FilterState holds the current filter configuration
type FilterState struct {
	Type         FilterType
	SearchQuery  string
	HostPatterns []HostPattern
	Methods      []string
	StatusCodes  []int
}

// NewFilterState creates a default filter state
func NewFilterState() *FilterState {
	return &FilterState{
		Type:         FilterAll,
		HostPatterns: []HostPattern{},
		Methods:      []string{},
		StatusCodes:  []int{},
	}
}

// Match checks if a flow matches the current filter
func (f *FilterState) Match(flow *Flow) bool {
	if flow == nil || flow.Request == nil {
		return false
	}

	// Check search query
	if f.SearchQuery != "" {
		query := strings.ToLower(f.SearchQuery)
		url := strings.ToLower(flow.Request.URL)
		host := strings.ToLower(flow.Request.Host)
		if !strings.Contains(url, query) && !strings.Contains(host, query) {
			return false
		}
	}

	// Check host patterns (only enabled ones)
	if len(f.HostPatterns) > 0 {
		hasEnabled := false
		matched := false
		for _, hp := range f.HostPatterns {
			if !hp.Enabled {
				continue
			}
			hasEnabled = true
			if matchPattern(flow.Request.Host, hp.Pattern) {
				matched = true
				break
			}
		}
		if hasEnabled {
			if f.Type == FilterWhitelist && !matched {
				return false
			}
			if f.Type == FilterCustom && !matched {
				return false
			}
		}
	}

	// Check methods
	if len(f.Methods) > 0 {
		methodMatch := false
		for _, m := range f.Methods {
			if flow.Request.Method == m {
				methodMatch = true
				break
			}
		}
		if !methodMatch {
			return false
		}
	}

	// Check status codes
	if len(f.StatusCodes) > 0 && flow.Response != nil {
		statusMatch := false
		for _, s := range f.StatusCodes {
			if flow.Response.StatusCode == s {
				statusMatch = true
				break
			}
		}
		if !statusMatch {
			return false
		}
	}

	return true
}

// matchPattern checks if host matches a glob-like pattern
func matchPattern(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	// Exact match
	if host == pattern {
		return true
	}

	// Simple glob matching with * wildcard
	if pattern == "*" {
		return true
	}

	// *.example.com matches example.com and sub.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		domain := pattern[2:] // "example.com"
		return host == domain || strings.HasSuffix(host, suffix)
	}

	// General glob pattern with * (e.g., *json.com, api.*, *google*)
	if strings.Contains(pattern, "*") {
		return matchGlob(host, pattern)
	}

	// Try regex if it looks like one
	if strings.ContainsAny(pattern, "^$()[]{}|+?\\") {
		re, err := regexp.Compile(pattern)
		if err == nil {
			return re.MatchString(host)
		}
	}

	return false
}

// matchGlob matches a string against a glob pattern with * wildcards
func matchGlob(s, pattern string) bool {
	// Convert glob to regex: escape special chars, replace * with .*
	regexPattern := "^"
	for _, c := range pattern {
		switch c {
		case '*':
			regexPattern += ".*"
		case '.', '+', '?', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
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
	return re.MatchString(s)
}
