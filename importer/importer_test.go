package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSimpleFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := `GET http://example.com/api
HTTP/1.1 200
`
	mainFile := filepath.Join(tmpDir, "main.hurlx")
	if err := os.WriteFile(mainFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir)
	resolved, err := r.Resolve(mainFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.File == nil {
		t.Fatal("expected non-nil file")
	}
	if len(resolved.File.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resolved.File.Entries))
	}
	if resolved.File.Entries[0].Request.Method != "GET" {
		t.Errorf("expected GET method, got %s", resolved.File.Entries[0].Request.Method)
	}
}

func TestResolveWithImport(t *testing.T) {
	tmpDir := t.TempDir()

	commonContent := `export base_url=http://example.com
`
	if err := os.WriteFile(filepath.Join(tmpDir, "common.hurlx"), []byte(commonContent), 0644); err != nil {
		t.Fatal(err)
	}

	mainContent := `import common
GET {{base_url}}/api
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.hurlx"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir)
	resolved, err := r.Resolve(filepath.Join(tmpDir, "main.hurlx"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.AllExports["base_url"] != "http://example.com" {
		t.Errorf("expected export base_url=http://example.com, got %q", resolved.AllExports["base_url"])
	}
	if len(resolved.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(resolved.Imports))
	}
}

func TestResolveCaching(t *testing.T) {
	tmpDir := t.TempDir()
	content := `GET http://example.com/api
`
	mainFile := filepath.Join(tmpDir, "main.hurlx")
	if err := os.WriteFile(mainFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir)
	r1, err := r.Resolve(mainFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, err := r.Resolve(mainFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1 != r2 {
		t.Error("expected same pointer from cache")
	}
}

func TestResolvePathTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	mainContent := `import ../../etc/passwd
GET http://example.com/api
`
	mainFile := filepath.Join(subDir, "main.hurlx")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir)
	_, err := r.Resolve(mainFile)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestResolveExtensionFallback(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "common.hurl"), []byte("GET http://example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mainContent := `import common
GET http://example.com/api
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.hurlx"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir)
	resolved, err := r.Resolve(filepath.Join(tmpDir, "main.hurlx"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(resolved.Imports))
	}
}
