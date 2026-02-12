package model

import (
	"strings"

	"proxy-tui/internal/util"
)

// FilterType represents different filter modes
type FilterType int

const (
	FilterAll FilterType = iota
	FilterWhitelist
	FilterCustom
	FilterStarred
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
	StarredIDs   map[FlowID]bool
}

// NewFilterState creates a default filter state
func NewFilterState() *FilterState {
	return &FilterState{
		Type:         FilterAll,
		HostPatterns: []HostPattern{},
		Methods:      []string{},
		StatusCodes:  []int{},
		StarredIDs:   make(map[FlowID]bool),
	}
}

// Match checks if a flow matches the current filter
func (f *FilterState) Match(flow *Flow) bool {
	if flow == nil || flow.Request == nil {
		return false
	}

	// Starred filter: only show starred flows
	if f.Type == FilterStarred {
		return f.StarredIDs[flow.ID]
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
			if util.MatchHostPattern(flow.Request.Host, hp.Pattern) {
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

