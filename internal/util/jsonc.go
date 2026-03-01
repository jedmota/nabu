package util

import (
	"bytes"
	"strings"
)

// StripJSONComments removes // line comments from JSONC data.
// It properly preserves // sequences inside JSON strings.
func StripJSONComments(data []byte) []byte {
	var result strings.Builder
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		inString := false
		i := 0
		for i < len(line) {
			c := line[i]

			if c == '"' && (i == 0 || line[i-1] != '\\') {
				inString = !inString
				result.WriteByte(c)
				i++
				continue
			}

			if !inString {
				if c == '/' && i+1 < len(line) && line[i+1] == '/' {
					break
				}
			}

			result.WriteByte(c)
			i++
		}
		result.WriteString("\n")
	}

	return []byte(result.String())
}

// IsJSON checks if data looks like JSON (starts with { or [).
func IsJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
