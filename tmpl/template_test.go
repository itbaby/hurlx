package tmpl

import (
	"testing"
)

func TestRenderVariable(t *testing.T) {
	vars := NewVariables()
	vars.Set("name", "world")
	result := Render("hello {{name}}!", vars)
	if result != "hello world!" {
		t.Errorf("expected 'hello world!', got %q", result)
	}
}

func TestRenderMultipleVariables(t *testing.T) {
	vars := NewVariables()
	vars.Set("host", "example.org")
	vars.Set("port", "8080")
	result := Render("https://{{host}}:{{port}}/api", vars)
	if result != "https://example.org:8080/api" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRenderNewUuid(t *testing.T) {
	vars := NewVariables()
	result := Render("{{newUuid}}", vars)
	if len(result) != 36 {
		t.Errorf("expected UUID length 36, got %d: %s", len(result), result)
	}
}

func TestRenderNewDate(t *testing.T) {
	vars := NewVariables()
	result := Render("{{newDate}}", vars)
	if len(result) < 20 {
		t.Errorf("expected date string, got %s", result)
	}
}

func TestRenderUndefined(t *testing.T) {
	vars := NewVariables()
	result := Render("{{undefined_var}}", vars)
	if result != "{{undefined_var}}" {
		t.Errorf("expected original template, got %s", result)
	}
}

func TestVariablesClone(t *testing.T) {
	vars := NewVariables()
	vars.Set("key", "value")
	clone := vars.Clone()
	clone.Set("key", "modified")
	if val, _ := vars.Get("key"); val != "value" {
		t.Error("clone should not affect original")
	}
}
