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
	FilterBlacklist
)

// FilterState holds the current filter configuration
type FilterState struct {
	Type         FilterType
	SearchQuery  string
	HostPatterns []string
	Methods      []string
	StatusCodes  []int
}

// NewFilterState creates a default filter state
func NewFilterState() *FilterState {
	return &FilterState{
		Type:         FilterAll,
		HostPatterns: []string{},
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

	// Check host patterns
	if len(f.HostPatterns) > 0 {
		matched := false
		for _, pattern := range f.HostPatterns {
			if matchPattern(flow.Request.Host, pattern) {
				matched = true
				break
			}
		}
		if f.Type == FilterWhitelist && !matched {
			return false
		}
		if f.Type == FilterBlacklist && matched {
			return false
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
	// Simple glob matching with * wildcard
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove the *
		return strings.HasSuffix(host, suffix) || host == pattern[2:]
	}
	// Try regex if it looks like one
	if strings.ContainsAny(pattern, "^$()[]{}|+?\\") {
		re, err := regexp.Compile(pattern)
		if err == nil {
			return re.MatchString(host)
		}
	}
	return host == pattern
}
