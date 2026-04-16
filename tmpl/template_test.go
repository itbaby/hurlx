package tmpl

import (
	"os"
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

func TestRenderUuid(t *testing.T) {
	vars := NewVariables()
	result := Render("{{uuid}}", vars)
	if len(result) != 36 {
		t.Errorf("expected UUID length 36, got %d: %s", len(result), result)
	}
	// Verify UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	if result[8] != '-' || result[13] != '-' || result[18] != '-' || result[23] != '-' {
		t.Errorf("expected UUID format, got %s", result)
	}
}

func TestRenderDate(t *testing.T) {
	vars := NewVariables()
	// Java-style format from README
	result := Render("{{date 'yyyy-MM-dd'}}", vars)
	if len(result) != 10 {
		t.Errorf("expected date length 10 (yyyy-MM-dd), got %d: %s", len(result), result)
	}
}

func TestRenderDateNoFormat(t *testing.T) {
	vars := NewVariables()
	result := Render("{{date}}", vars)
	if len(result) < 20 {
		t.Errorf("expected date string, got %s", result)
	}
}

func TestRenderRandomHex(t *testing.T) {
	vars := NewVariables()
	result := Render("{{randomHex 16}}", vars)
	if len(result) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("expected hex length 32, got %d: %s", len(result), result)
	}
}

func TestRenderRandomHexDefault(t *testing.T) {
	vars := NewVariables()
	result := Render("{{randomHex}}", vars)
	if len(result) != 64 { // default 32 bytes = 64 hex chars
		t.Errorf("expected hex length 64, got %d: %s", len(result), result)
	}
}

func TestRenderGetenvLowercase(t *testing.T) {
	vars := NewVariables()
	result := Render("{{getenv 'HOME'}}", vars)
	home := os.Getenv("HOME")
	if result != home {
		t.Errorf("expected %q, got %q", home, result)
	}
}
