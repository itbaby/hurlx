package parser

import (
	"testing"
)

func TestParseSimpleGet(t *testing.T) {
	input := `GET https://example.org
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(file.Entries))
	}
	if file.Entries[0].Request.Method != "GET" {
		t.Errorf("expected GET, got %s", file.Entries[0].Request.Method)
	}
	if file.Entries[0].Request.URL != "https://example.org" {
		t.Errorf("expected https://example.org, got %s", file.Entries[0].Request.URL)
	}
	if file.Entries[0].Response.Status != 200 {
		t.Errorf("expected status 200, got %d", file.Entries[0].Response.Status)
	}
}

func TestParsePostJSON(t *testing.T) {
	input := `POST https://example.org/api/dogs
Content-Type: application/json
{
    "name": "Frieda",
    "age": 3
}
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(file.Entries))
	}
	entry := file.Entries[0]
	if entry.Request.Method != "POST" {
		t.Errorf("expected POST, got %s", entry.Request.Method)
	}
	if len(entry.Request.Headers) < 1 {
		t.Errorf("expected at least 1 header")
	}
	if entry.Request.Body == nil {
		t.Fatal("expected body")
	}
	if entry.Request.Body.Type != 1 { // BodyJSON
		t.Errorf("expected JSON body type")
	}
}

func TestParseImport(t *testing.T) {
	input := `import "common.hurlx"
import "auth.hurlx" as auth

GET https://example.org/api/test
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(file.Imports))
	}
	if file.Imports[0].Path != "common.hurlx" {
		t.Errorf("expected common.hurlx, got %s", file.Imports[0].Path)
	}
	if file.Imports[0].Alias != "" {
		t.Errorf("expected empty alias, got %s", file.Imports[0].Alias)
	}
	if file.Imports[1].Path != "auth.hurlx" {
		t.Errorf("expected auth.hurlx, got %s", file.Imports[1].Path)
	}
	if file.Imports[1].Alias != "auth" {
		t.Errorf("expected auth alias, got %s", file.Imports[1].Alias)
	}
}

func TestParseExport(t *testing.T) {
	input := `export base_url = https://api.example.org
export timeout = 30

GET {{base_url}}/health
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Exports) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(file.Exports))
	}
	if file.Exports[0].Name != "base_url" {
		t.Errorf("expected base_url, got %s", file.Exports[0].Name)
	}
	if file.Exports[0].Value != "https://api.example.org" {
		t.Errorf("expected https://api.example.org, got %s", file.Exports[0].Value)
	}
}

func TestParseQueryParams(t *testing.T) {
	input := `GET https://example.org/api/search
[Query]
q: hurlx
page: 1
limit: 10
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	entry := file.Entries[0]
	if len(entry.Request.Query) != 3 {
		t.Fatalf("expected 3 query params, got %d", len(entry.Request.Query))
	}
	if entry.Request.Query[0].Key != "q" || entry.Request.Query[0].Value != "hurlx" {
		t.Errorf("unexpected query param: %v", entry.Request.Query[0])
	}
}

func TestParseFormParams(t *testing.T) {
	input := `POST https://example.org/login
[Form]
username: admin
password: secret
HTTP 302`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	entry := file.Entries[0]
	if len(entry.Request.Form) != 2 {
		t.Fatalf("expected 2 form params, got %d", len(entry.Request.Form))
	}
	if entry.Response.Status != 302 {
		t.Errorf("expected status 302, got %d", entry.Response.Status)
	}
}

func TestParseCaptures(t *testing.T) {
	input := `GET https://example.org/api/token
HTTP 200
[Captures]
token: jsonpath "$.access_token"
user_id: header "X-User-Id"

GET https://example.org/api/user/{{user_id}}
Authorization: Bearer {{token}}
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(file.Entries))
	}
	firstResp := file.Entries[0].Response
	if len(firstResp.Captures) != 2 {
		t.Fatalf("expected 2 captures, got %d", len(firstResp.Captures))
	}
	if firstResp.Captures[0].Variable != "token" {
		t.Errorf("expected token capture, got %s", firstResp.Captures[0].Variable)
	}
}

func TestParseAsserts(t *testing.T) {
	input := `GET https://example.org/api/status
HTTP 200
[Asserts]
jsonpath "$.status" == "running"
jsonpath "$.count" >= 10
header "Content-Type" contains "json"
duration < 1000`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 4 {
		t.Fatalf("expected 4 asserts, got %d", len(asserts))
	}
}

func TestParseChainingRequests(t *testing.T) {
	input := `GET https://example.org/step1
HTTP 200

GET https://example.org/step2
HTTP 200

GET https://example.org/step3
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(file.Entries))
	}
}

func TestParseBasicAuth(t *testing.T) {
	input := `GET https://example.org/protected
[BasicAuth]
admin: secret
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	auth := file.Entries[0].Request.BasicAuth
	if auth == nil {
		t.Fatal("expected basic auth")
	}
	if auth.Username != "admin" || auth.Password != "secret" {
		t.Errorf("unexpected auth: %s:%s", auth.Username, auth.Password)
	}
}

func TestParseComments(t *testing.T) {
	input := `# This is a comment
GET https://example.org
# Another comment
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Comments) < 1 {
		t.Fatalf("expected comments")
	}
}

func TestParseOptions(t *testing.T) {
	input := `GET https://example.org
[Options]
location: true
retry: 3
verbose: true
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	opts := file.Entries[0].Request.Options
	if opts == nil {
		t.Fatal("expected options")
	}
	if opts.Location == nil || !*opts.Location {
		t.Error("expected location true")
	}
	if opts.Retry == nil || *opts.Retry != 3 {
		t.Error("expected retry 3")
	}
}

func TestParseMultilineBody(t *testing.T) {
	input := "POST https://example.org/upload\nContent-Type: text/csv\n```\nname,age\nAlice,30\n```\nHTTP 200"

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	body := file.Entries[0].Request.Body
	if body == nil {
		t.Fatal("expected body")
	}
	if body.Type != 3 { // BodyMultiline
		t.Errorf("expected multiline body, got %d", body.Type)
	}
}
