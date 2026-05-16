package parser

import (
	"testing"

	"github.com/wei-lli/hurlx/ast"
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

func TestParseAssertsWithBlankLines(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Asserts]
status == 200

jsonpath "$.name" == "test"

jsonpath "$.count" > 0

header "Content-Type" contains "json"

duration < 1000`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 5 {
		t.Fatalf("expected 5 asserts, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QueryStatus {
		t.Errorf("expected QueryStatus, got %d", asserts[0].Query.Type)
	}
	if asserts[2].Value.Int != 0 {
		t.Errorf("expected count > 0 value")
	}
	if asserts[4].Query.Type != ast.QueryDuration {
		t.Errorf("expected QueryDuration, got %d", asserts[4].Query.Type)
	}
}

func TestParseCapturesWithBlankLines(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Captures]
token: header "X-Auth-Token"

user_id: jsonpath "$.id"

name: jsonpath "$.name" regex "Mr (.*)"

body_content: body`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	captures := file.Entries[0].Response.Captures
	if len(captures) != 4 {
		t.Fatalf("expected 4 captures, got %d", len(captures))
	}
	if captures[0].Variable != "token" {
		t.Errorf("expected token, got %s", captures[0].Variable)
	}
	if captures[3].Variable != "body_content" {
		t.Errorf("expected body_content, got %s", captures[3].Variable)
	}
}

func TestParseQueryWithBlankLines(t *testing.T) {
	input := `GET https://example.org
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
	if entry.Request.Query[2].Key != "limit" || entry.Request.Query[2].Value != "10" {
		t.Errorf("expected limit=10, got %s=%s", entry.Request.Query[2].Key, entry.Request.Query[2].Value)
	}
}

func TestParseFormWithBlankLines(t *testing.T) {
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

func TestParseMultipartWithBlankLines(t *testing.T) {
	input := `POST https://example.org/upload
[Multipart]
name: file.txt

file1: file,data.bin;

file2: file,image.png; image/png
HTTP 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if len(req.Multipart) != 3 {
		t.Fatalf("expected 3 multipart fields, got %d", len(req.Multipart))
	}
	if req.Multipart[1].IsFile != true {
		t.Error("expected file field")
	}
}

func TestParseOptionsWithBlankLines(t *testing.T) {
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

func TestParseResponseWithBlankLines(t *testing.T) {
	input := `GET https://example.org
HTTP 200
Content-Type: application/json

[Asserts]
status == 200`

	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Status != 200 {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
	if len(resp.Headers) != 1 {
		t.Fatalf("expected 1 response header, got %d", len(resp.Headers))
	}
	if len(resp.Asserts) != 1 {
		t.Fatalf("expected 1 assert, got %d", len(resp.Asserts))
	}
}

func TestParseMultipleBlankLinesBetweenEntries(t *testing.T) {
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
