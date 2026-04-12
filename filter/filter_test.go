package filter

import (
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
