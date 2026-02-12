package util

import (
	"regexp"
	"strings"
)

// MatchGlob matches a string against a glob pattern with * and ? wildcards.
// Consolidates matchGlob (proxy, filter) and matchGlobPattern (mapping).
func MatchGlob(s, pattern string) bool {
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
	return re.MatchString(s)
}

// MatchHostPattern checks if host matches a glob-like pattern.
// Supports exact match, *, *.domain, general glob, and regex fallback.
// Strips port from host if present.
// Consolidates matchHostPattern (proxy) and matchPattern (filter).
func MatchHostPattern(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard matching
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
		return MatchGlob(host, pattern)
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
