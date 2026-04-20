package filter

import (
	"encoding/base64"
	"testing"

	"github.com/wei-lli/hurlx/ast"
)

func TestExtractJSONPath(t *testing.T) {
	data := []byte(`{"name": "Alice", "age": 30, "address": {"city": "Beijing"}}`)

	tests := []struct {
		path     string
		expected interface{}
	}{
		{"$.name", "Alice"},
		{"$.age", float64(30)},
		{"$.address.city", "Beijing"},
	}

	for _, tt := range tests {
		result, err := ExtractJSONPath(data, tt.path)
		if err != nil {
			t.Errorf("jsonpath %s error: %v", tt.path, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("jsonpath %s: expected %v (%T), got %v (%T)", tt.path, tt.expected, tt.expected, result, result)
		}
	}
}

func TestExtractJSONPathArray(t *testing.T) {
	data := []byte(`{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`)

	result, err := ExtractJSONPath(data, "$.items[0].id")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result for $.items[0].id")
	}
	if result != float64(1) {
		t.Errorf("expected 1, got %v", result)
	}
}

func TestFilterCount(t *testing.T) {
	input := []interface{}{"a", "b", "c"}
	result, err := Apply(input, []ast.Filter{{Type: ast.FilterCount}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestFilterSplit(t *testing.T) {
	input := "a,b,c"
	result, err := Apply(input, []ast.Filter{{Type: ast.FilterSplit, Value: ","}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 items, got %d", len(arr))
	}
}

func TestFilterNth(t *testing.T) {
	input := []interface{}{"a", "b", "c"}
	result, err := Apply(input, []ast.Filter{{Type: ast.FilterNth, Value: "1"}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "b" {
		t.Errorf("expected 'b', got %v", result)
	}
}

func TestFilterToInt(t *testing.T) {
	result, err := Apply("42", []ast.Filter{{Type: ast.FilterToInt}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestFilterToString(t *testing.T) {
	result, err := Apply(42, []ast.Filter{{Type: ast.FilterToString}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "42" {
		t.Errorf("expected '42', got %v", result)
	}
}

func TestBase64RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hello", "Hello, World!"},
		{"empty", ""},
		{"one byte", "A"},
		{"two bytes", "AB"},
		{"three bytes", "ABC"},
		{"json", `{"key":"value"}`},
		{"binary", "\x00\x01\x02\xff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeBase64([]byte(tt.input))
			decoded, err := decodeBase64(encoded)
			if err != nil {
				t.Fatalf("decodeBase64(%q) error: %v", encoded, err)
			}
			if string(decoded) != tt.input {
				t.Errorf("round-trip failed: input=%q, encoded=%q, decoded=%q", tt.input, encoded, decoded)
			}
		})
	}
}

func TestBase64DecodePadded(t *testing.T) {
	// Users commonly provide padded base64 input
	tests := []struct {
		input    string
		expected string
	}{
		{"SGVsbG8=", "Hello"},
		{"QVBC", "APB"},
		{"AA==", "\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			decoded, err := decodeBase64(tt.input)
			if err != nil {
				t.Fatalf("decodeBase64(%q) error: %v", tt.input, err)
			}
			if string(decoded) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, decoded)
			}
		})
	}
}

func TestBase64UrlSafeRoundTrip(t *testing.T) {
	input := "Hello, World!"
	encoded := encodeBase64URLSafe([]byte(input))
	decoded, err := decodeBase64URLSafe(encoded)
	if err != nil {
		t.Fatalf("decodeBase64URLSafe error: %v", err)
	}
	if string(decoded) != input {
		t.Errorf("round-trip failed: input=%q, encoded=%q, decoded=%q", input, encoded, decoded)
	}
}

func TestBase64UrlSafeDecodePadded(t *testing.T) {
	// Padded URL-safe base64 input (P2-2: TrimRight fix)
	input := "Hello, World!"
	padded := base64.URLEncoding.EncodeToString([]byte(input)) // produces padded output
	decoded, err := decodeBase64URLSafe(padded)
	if err != nil {
		t.Fatalf("decodeBase64URLSafe with padding error: %v", err)
	}
	if string(decoded) != input {
		t.Errorf("padded decode failed: input=%q, padded=%q, decoded=%q", input, padded, decoded)
	}
}

func TestFilterChaining(t *testing.T) {
	input := "a,b,c,d"
	result, err := Apply(input, []ast.Filter{
		{Type: ast.FilterSplit, Value: ","},
		{Type: ast.FilterCount},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 4 {
		t.Errorf("expected 4, got %v", result)
	}
}
