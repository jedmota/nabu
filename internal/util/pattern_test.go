package util

import "testing"

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		pattern string
		want    bool
	}{
		{"star matches all", "anything", "*", true},
		{"star prefix", "hello world", "hello*", true},
		{"star suffix", "hello world", "*world", true},
		{"star middle", "hello world", "he*ld", true},
		{"question mark", "abc", "a?c", true},
		{"question mark no match", "abcd", "a?c", false},
		{"dot escaped", "example.com", "example.com", true},
		{"dot not wildcard", "exampleXcom", "example.com", false},
		{"complex glob", "api.google.com", "*google*", true},
		{"no match", "other.com", "example*", false},
		{"empty pattern", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchGlob(tt.s, tt.pattern)
			if got != tt.want {
				t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchHostPattern(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		pattern string
		want    bool
	}{
		{"exact", "example.com", "example.com", true},
		{"wildcard star", "anything.com", "*", true},
		{"wildcard subdomain", "api.example.com", "*.example.com", true},
		{"wildcard subdomain bare domain", "example.com", "*.example.com", true},
		{"general glob", "api.google.com", "*google*", true},
		{"regex", "api.example.com", "^api\\.", true},
		{"case insensitive", "Example.COM", "example.com", true},
		{"port stripping", "example.com:443", "example.com", true},
		{"port stripping with wildcard", "api.example.com:8080", "*.example.com", true},
		{"no match", "other.com", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchHostPattern(tt.host, tt.pattern)
			if got != tt.want {
				t.Errorf("MatchHostPattern(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}
