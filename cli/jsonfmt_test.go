package main

import (
	"strings"
	"testing"
)

func TestIsJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`  {"key": "value"}  `, true},
		{`  [1, 2, 3]  `, true},
		{`plain text`, false},
		{`<html></html>`, false},
		{``, false},
		{`   `, false},
		{`null`, false},
		{`42`, false},
		{`"hello"`, false},
	}

	for _, tc := range tests {
		result := isJSON([]byte(tc.input))
		if result != tc.expected {
			t.Errorf("isJSON(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`{"nested": {"a": 1}}`, true},
		{`[{"a": 1}, {"b": 2}]`, true},
		{`{invalid}`, false},
		{`[1, 2,`, false},
		{`plain text`, false},
		{``, false},
		{`null`, false},
	}

	for _, tc := range tests {
		result := isValidJSON([]byte(tc.input))
		if result != tc.expected {
			t.Errorf("isValidJSON(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestPrettifyJSON(t *testing.T) {
	input := `{"name":"test","value":42,"active":true,"items":[1,2,3]}`
	expected := `{
  "active": true,
  "items": [
    1,
    2,
    3
  ],
  "name": "test",
  "value": 42
}`

	result, err := prettifyJSON([]byte(input))
	if err != nil {
		t.Fatalf("prettifyJSON returned error: %v", err)
	}
	if string(result) != expected {
		t.Errorf("prettifyJSON output mismatch.\nGot:\n%s\nWant:\n%s", string(result), expected)
	}
}

func TestPrettifyJSON_AlreadyPretty(t *testing.T) {
	input := "{\n  \"key\": \"value\"\n}"
	result, err := prettifyJSON([]byte(input))
	if err != nil {
		t.Fatalf("prettifyJSON returned error: %v", err)
	}
	if string(result) != input {
		t.Errorf("prettifyJSON changed already-pretty JSON.\nGot:\n%s\nWant:\n%s", string(result), input)
	}
}

func TestPrettifyJSON_InvalidJSON(t *testing.T) {
	_, err := prettifyJSON([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestColorizeJSON_Keys(t *testing.T) {
	input := `{
  "name": "test"
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[36m") {
		t.Error("expected cyan color for key")
	}
	if !strings.Contains(result, "\x1b[32m") {
		t.Error("expected green color for string value")
	}
}

func TestColorizeJSON_Numbers(t *testing.T) {
	input := `{
  "count": 42,
  "ratio": 3.14
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[33m") {
		t.Error("expected yellow color for numbers")
	}
}

func TestColorizeJSON_Booleans(t *testing.T) {
	input := `{
  "active": true,
  "deleted": false
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[35mtrue") {
		t.Error("expected magenta color for true")
	}
	if !strings.Contains(result, "\x1b[35mfalse") {
		t.Error("expected magenta color for false")
	}
}

func TestColorizeJSON_Null(t *testing.T) {
	input := `{
  "value": null
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[90mnull") {
		t.Error("expected gray color for null")
	}
}

func TestColorizeJSON_Nested(t *testing.T) {
	input := `{
  "user": {
    "name": "Alice",
    "age": 30
  },
  "tags": ["a", "b"]
}`
	result := string(colorizeJSON([]byte(input)))

	keyCount := strings.Count(result, "\x1b[36m\"")
	if keyCount < 4 {
		t.Errorf("expected at least 4 colored keys, got %d", keyCount)
	}

	if !strings.Contains(result, "\x1b[33m") {
		t.Error("expected yellow color for number")
	}
}

func TestColorizeJSON_Array(t *testing.T) {
	input := `[
  1,
  "hello",
  true,
  null,
  {"key": "value"}
]`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[33m1") {
		t.Error("expected yellow for number in array")
	}
	if !strings.Contains(result, "\x1b[32m\"hello\"") {
		t.Error("expected green for string in array")
	}
	if !strings.Contains(result, "\x1b[35mtrue") {
		t.Error("expected magenta for boolean in array")
	}
	if !strings.Contains(result, "\x1b[90mnull") {
		t.Error("expected gray for null in array")
	}
}

func TestColorizeJSON_EscapedStrings(t *testing.T) {
	input := `{
  "path": "C:\\Users\\test",
  "quote": "He said \"hello\""
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[32m") {
		t.Error("expected green color for string values with escapes")
	}
}

func TestColorizeJSON_NegativeNumbers(t *testing.T) {
	input := `{
  "temp": -10,
  "offset": -3.5
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[33m-10") {
		t.Error("expected yellow for negative integer")
	}
	if !strings.Contains(result, "\x1b[33m-3.5") {
		t.Error("expected yellow for negative float")
	}
}

func TestColorizeJSON_ScientificNotation(t *testing.T) {
	input := `{
  "big": 1.5e10,
  "small": 2.3E-4
}`
	result := string(colorizeJSON([]byte(input)))

	if !strings.Contains(result, "\x1b[33m1.5e10") {
		t.Error("expected yellow for scientific notation")
	}
	if !strings.Contains(result, "\x1b[33m2.3E-4") {
		t.Error("expected yellow for scientific notation with capital E")
	}
}

func TestColorizeJSON_EmptyObject(t *testing.T) {
	input := `{}`
	result := string(colorizeJSON([]byte(input)))
	if result != `{}` {
		t.Errorf("unexpected output for empty object: %s", result)
	}
}

func TestColorizeJSON_EmptyArray(t *testing.T) {
	input := `[]`
	result := string(colorizeJSON([]byte(input)))
	if result != `[]` {
		t.Errorf("unexpected output for empty array: %s", result)
	}
}

func TestIsKey(t *testing.T) {
	tests := []struct {
		input string
		pos   int
		want  bool
	}{
		{`{"key": "value"}`, 2, true},
		{`{"key": "value"}`, 9, false},
		{`{"a":1,"b":2}`, 7, true},
		{`{"a":1,"b":2}`, 1, true},
	}

	for _, tc := range tests {
		got := isKey([]byte(tc.input), tc.pos)
		if got != tc.want {
			t.Errorf("isKey(%q, %d) = %v, want %v", tc.input, tc.pos, got, tc.want)
		}
	}
}

func TestIsKey_WithWhitespace(t *testing.T) {
	input := "{ \"name\" : \"test\" }"
	got := isKey([]byte(input), 2)
	if !got {
		t.Error("expected true for key after '{' with whitespace")
	}
}
