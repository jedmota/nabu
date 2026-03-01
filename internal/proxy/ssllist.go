package proxy

import (
	"sync"

	"nabu/internal/util"
)

// SSLProxyList manages the list of hosts to perform MITM on
type SSLProxyList struct {
	patterns []string
	mu       sync.RWMutex
}

// NewSSLProxyList creates a new SSL proxy list
func NewSSLProxyList() *SSLProxyList {
	return &SSLProxyList{
		patterns: make([]string, 0),
	}
}

// Add adds a pattern to the list
func (s *SSLProxyList) Add(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check if already exists
	for _, p := range s.patterns {
		if p == pattern {
			return
		}
	}
	s.patterns = append(s.patterns, pattern)
}

// Remove removes a pattern from the list
func (s *SSLProxyList) Remove(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.patterns {
		if p == pattern {
			s.patterns = append(s.patterns[:i], s.patterns[i+1:]...)
			return
		}
	}
}

// Clear removes all patterns
func (s *SSLProxyList) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patterns = make([]string, 0)
}

// Patterns returns a copy of all patterns
func (s *SSLProxyList) Patterns() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.patterns))
	copy(result, s.patterns)
	return result
}

// Match checks if a host matches any pattern in the list
func (s *SSLProxyList) Match(host string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If list is empty, don't match anything (passthrough all)
	if len(s.patterns) == 0 {
		return false
	}

	for _, pattern := range s.patterns {
		if util.MatchHostPattern(host, pattern) {
			return true
		}
	}
	return false
}
