package util

import (
	"strings"
	"testing"
)

func TestStripJSONComments_LineComment(t *testing.T) {
	input := "// comment\n{\"key\": \"value\"}\n"
	got := string(StripJSONComments([]byte(input)))
	if strings.Contains(got, "// comment") {
		t.Error("line comment should be stripped")
	}
	if !strings.Contains(got, `"key": "value"`) {
		t.Error("JSON content should be preserved")
	}
}

func TestStripJSONComments_InlineComment(t *testing.T) {
	input := "{\n  \"key\": \"value\", // inline comment\n}\n"
	got := string(StripJSONComments([]byte(input)))
	if strings.Contains(got, "// inline comment") {
		t.Error("inline comment should be stripped")
	}
	if !strings.Contains(got, `"key": "value"`) {
		t.Error("values should be preserved")
	}
}

func TestStripJSONComments_InsideString(t *testing.T) {
	input := `{"url": "http://example.com//path"}` + "\n"
	got := string(StripJSONComments([]byte(input)))
	if !strings.Contains(got, "http://example.com//path") {
		t.Error("// inside JSON string should be preserved")
	}
}
