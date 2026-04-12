package parser

import (
	"testing"

	"github.com/wei-lli/hurlx/ast"
)

func TestHurlGrammar_BasicGet(t *testing.T) {
	input := `GET https://example.org`
	p := NewParser(input, "test.hurl")
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
}

func TestHurlGrammar_StatusWildcard(t *testing.T) {
	input := `GET https://example.org
HTTP *`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Status != 0 {
		t.Errorf("expected wildcard status 0, got %d", resp.Status)
	}
}

func TestHurlGrammar_ExplicitVersion(t *testing.T) {
	input := `GET https://example.org/api/pets
HTTP/2 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Version != "2" {
		t.Errorf("expected version 2, got %s", resp.Version)
	}
	if resp.Status != 200 {
		t.Errorf("expected status 200, got %d", resp.Status)
	}
}

func TestHurlGrammar_Headers(t *testing.T) {
	input := `GET https://example.org/api/cats
Content-Type: application/json
Authorization: Bearer {{token}}
HTTP 200
Content-Type: application/json; charset=utf-8`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if len(req.Headers) != 2 {
		t.Fatalf("expected 2 request headers, got %d", len(req.Headers))
	}
	if req.Headers[0].Name != "Content-Type" {
		t.Errorf("expected Content-Type, got %s", req.Headers[0].Name)
	}
	if req.Headers[1].Value != "Bearer {{token}}" {
		t.Errorf("expected Bearer {{token}}, got %s", req.Headers[1].Value)
	}
	resp := file.Entries[0].Response
	if len(resp.Headers) != 1 {
		t.Fatalf("expected 1 response header, got %d", len(resp.Headers))
	}
	if resp.Headers[0].Name != "Content-Type" {
		t.Errorf("expected Content-Type, got %s", resp.Headers[0].Name)
	}
}

func TestHurlGrammar_QuerySection(t *testing.T) {
	input := `GET https://example.org/api
[Query]
page: 1
size: 20
filter: active
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if len(req.Query) != 3 {
		t.Fatalf("expected 3 query params, got %d", len(req.Query))
	}
	if req.Query[0].Key != "page" || req.Query[0].Value != "1" {
		t.Errorf("expected page=1, got %s=%s", req.Query[0].Key, req.Query[0].Value)
	}
}

func TestHurlGrammar_FormSection(t *testing.T) {
	input := `POST https://example.org/login
[Form]
user: toto
password: 12345678
HTTP 302`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if req.Method != "POST" {
		t.Errorf("expected POST, got %s", req.Method)
	}
	if len(req.Form) != 2 {
		t.Fatalf("expected 2 form params, got %d", len(req.Form))
	}
}

func TestHurlGrammar_MultipartSection(t *testing.T) {
	input := `POST https://example.org/upload
[Multipart]
name: file.txt
file1: file,data.bin;
file2: file,image.png; image/png
HTTP 200`
	p := NewParser(input, "test.hurl")
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
	if req.Multipart[2].FileType != "image/png" {
		t.Errorf("expected image/png, got %s", req.Multipart[2].FileType)
	}
}

func TestHurlGrammar_BasicAuthSection(t *testing.T) {
	input := `GET https://example.org/protected
[BasicAuth]
user: pass
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ba := file.Entries[0].Request.BasicAuth
	if ba == nil {
		t.Fatal("expected BasicAuth")
	}
	if ba.Username != "user" || ba.Password != "pass" {
		t.Errorf("expected user:pass, got %s:%s", ba.Username, ba.Password)
	}
}

func TestHurlGrammar_OptionsSection(t *testing.T) {
	input := `GET https://example.org/api
[Options]
location: true
max-redirs: 10
insecure: true
timeout: 30s
delay: 500ms
retry: 3
retry-interval: 1s
skip: false
verbose: true
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	opts := file.Entries[0].Request.Options
	if opts == nil {
		t.Fatal("expected Options section")
	}
	if opts.Location == nil || !*opts.Location {
		t.Error("expected location true")
	}
	if opts.MaxRedirs == nil || *opts.MaxRedirs != 10 {
		t.Error("expected max-redirs 10")
	}
	if opts.Timeout != "30s" {
		t.Errorf("expected timeout 30s, got %s", opts.Timeout)
	}
	if opts.Retry == nil || *opts.Retry != 3 {
		t.Error("expected retry 3")
	}
}

func TestHurlGrammar_CookiesSection(t *testing.T) {
	input := `GET https://example.org
[Cookies]
session: abc123
theme: dark
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if len(req.Cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(req.Cookies))
	}
}

func TestHurlGrammar_Captures(t *testing.T) {
	input := `POST https://example.org/login
[Form]
user: admin
password: admin
HTTP 200
[Captures]
token: header "X-Auth-Token"
user_id: jsonpath "$.id"
name: jsonpath "$.name" regex "Mr (.*)"
body_content: body`
	p := NewParser(input, "test.hurl")
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
	if captures[0].Query.Type != ast.QueryHeader {
		t.Errorf("expected QueryHeader, got %d", captures[0].Query.Type)
	}
	if captures[0].Query.Value != "X-Auth-Token" {
		t.Errorf("expected X-Auth-Token, got %s", captures[0].Query.Value)
	}
	if captures[1].Query.Type != ast.QueryJSONPath {
		t.Errorf("expected QueryJSONPath, got %d", captures[1].Query.Type)
	}
	if captures[2].Filters == nil || len(captures[2].Filters) != 1 {
		t.Fatal("expected 1 filter on capture 2")
	}
	if captures[2].Filters[0].Type != ast.FilterRegex {
		t.Errorf("expected FilterRegex, got %d", captures[2].Filters[0].Type)
	}
}

func TestHurlGrammar_Asserts(t *testing.T) {
	input := `GET https://example.org/api/cats
HTTP 200
[Asserts]
status == 200
header "Content-Type" contains "json"
jsonpath "$.cats" count == 49
jsonpath "$.cats[0].name" == "Felix"
jsonpath "$.cats[0].lives" == 9
jsonpath "$.active" not exists
body contains "Felix"
bytes count == 120`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 8 {
		t.Fatalf("expected 8 asserts, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QueryStatus {
		t.Errorf("expected QueryStatus, got %d", asserts[0].Query.Type)
	}
	if asserts[0].Predicate != ast.PredEqual {
		t.Errorf("expected PredEqual, got %d", asserts[0].Predicate)
	}
	if asserts[5].Not != true {
		t.Error("expected not flag on assert 5")
	}
	if asserts[5].Predicate != ast.PredExists {
		t.Errorf("expected PredExists, got %d", asserts[5].Predicate)
	}
}

func TestHurlGrammar_AllPredicateTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		predType ast.PredicateType
		not      bool
	}{
		{"equal", `jsonpath "$.x" == 1`, ast.PredEqual, false},
		{"not_equal", `jsonpath "$.x" != 1`, ast.PredNotEqual, false},
		{"greater", `jsonpath "$.x" > 1`, ast.PredGreaterThan, false},
		{"greater_equal", `jsonpath "$.x" >= 1`, ast.PredGreaterEqual, false},
		{"less", `jsonpath "$.x" < 1`, ast.PredLessThan, false},
		{"less_equal", `jsonpath "$.x" <= 1`, ast.PredLessEqual, false},
		{"starts_with", `jsonpath "$.x" startsWith "foo"`, ast.PredStartsWith, false},
		{"ends_with", `jsonpath "$.x" endsWith "foo"`, ast.PredEndsWith, false},
		{"contains", `jsonpath "$.x" contains "foo"`, ast.PredContains, false},
		{"matches_regex", `jsonpath "$.x" matches "\\d+"`, ast.PredMatches, false},
		{"exists", `jsonpath "$.x" exists`, ast.PredExists, false},
		{"not_exists", `jsonpath "$.x" not exists`, ast.PredExists, true},
		{"isBoolean", `jsonpath "$.x" isBoolean`, ast.PredIsBoolean, false},
		{"isEmpty", `jsonpath "$.x" isEmpty`, ast.PredIsEmpty, false},
		{"isFloat", `jsonpath "$.x" isFloat`, ast.PredIsFloat, false},
		{"isInteger", `jsonpath "$.x" isInteger`, ast.PredIsInteger, false},
		{"isIpv4", `ip isIpv4`, ast.PredIsIpv4, false},
		{"isIpv6", `ip isIpv6`, ast.PredIsIpv6, false},
		{"isIsoDate", `jsonpath "$.x" isIsoDate`, ast.PredIsIsoDate, false},
		{"isList", `jsonpath "$.x" isList`, ast.PredIsList, false},
		{"isNumber", `jsonpath "$.x" isNumber`, ast.PredIsNumber, false},
		{"isObject", `jsonpath "$.x" isObject`, ast.PredIsObject, false},
		{"isString", `jsonpath "$.x" isString`, ast.PredIsString, false},
		{"isUuid", `jsonpath "$.x" isUuid`, ast.PredIsUuid, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "GET https://example.org\nHTTP 200\n[Asserts]\n" + tt.input
			p := NewParser(input, "test.hurl")
			file, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			asserts := file.Entries[0].Response.Asserts
			if len(asserts) != 1 {
				t.Fatalf("expected 1 assert, got %d", len(asserts))
			}
			if asserts[0].Predicate != tt.predType {
				t.Errorf("expected predicate %d, got %d", tt.predType, asserts[0].Predicate)
			}
			if asserts[0].Not != tt.not {
				t.Errorf("expected not=%v, got %v", tt.not, asserts[0].Not)
			}
		})
	}
}

func TestHurlGrammar_AllQueryTypes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		qtype  ast.QueryType
		qvalue string
	}{
		{"status", "status == 200", ast.QueryStatus, ""},
		{"version", `version == "1.1"`, ast.QueryVersion, ""},
		{"url", `url == "https://example.org"`, ast.QueryURL, ""},
		{"header", `header "Content-Type" contains "json"`, ast.QueryHeader, "Content-Type"},
		{"cookie", `cookie "session" == "abc"`, ast.QueryCookie, "session"},
		{"body", `body contains "hello"`, ast.QueryBody, ""},
		{"bytes", `bytes count == 100`, ast.QueryBytes, ""},
		{"xpath", `xpath "//h1" exists`, ast.QueryXPath, "//h1"},
		{"jsonpath", `jsonpath "$.name" == "test"`, ast.QueryJSONPath, "$.name"},
		{"regex", `regex "^(\\d+)$" == "123"`, ast.QueryRegex, "^(\\\\d+)$"},
		{"variable", `variable "x" == "y"`, ast.QueryVariable, "x"},
		{"duration", "duration < 1000", ast.QueryDuration, ""},
		{"sha256", `sha256 == hex,abcdef;`, ast.QuerySHA256, ""},
		{"md5", `md5 == hex,abcdef;`, ast.QueryMD5, ""},
		{"redirects", "redirects count == 1", ast.QueryRedirects, ""},
		{"ip", `ip isIpv4`, ast.QueryIP, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "GET https://example.org\nHTTP 200\n[Asserts]\n" + tt.input
			p := NewParser(input, "test.hurl")
			file, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			asserts := file.Entries[0].Response.Asserts
			if len(asserts) != 1 {
				t.Fatalf("expected 1 assert, got %d", len(asserts))
			}
			if asserts[0].Query.Type != tt.qtype {
				t.Errorf("expected query type %d, got %d", tt.qtype, asserts[0].Query.Type)
			}
			if tt.qvalue != "" && asserts[0].Query.Value != tt.qvalue {
				t.Errorf("expected query value %q, got %q", tt.qvalue, asserts[0].Query.Value)
			}
		})
	}
}

func TestHurlGrammar_FilterChaining(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Captures]
items: jsonpath "$.items" split "," nth 0`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	captures := file.Entries[0].Response.Captures
	if len(captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(captures))
	}
	if len(captures[0].Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(captures[0].Filters))
	}
	if captures[0].Filters[0].Type != ast.FilterSplit {
		t.Errorf("expected FilterSplit, got %d", captures[0].Filters[0].Type)
	}
	if captures[0].Filters[0].Value != "," {
		t.Errorf("expected comma, got %s", captures[0].Filters[0].Value)
	}
	if captures[0].Filters[1].Type != ast.FilterNth {
		t.Errorf("expected FilterNth, got %d", captures[0].Filters[1].Type)
	}
	if captures[0].Filters[1].Value != "0" {
		t.Errorf("expected 0, got %s", captures[0].Filters[1].Value)
	}
}

func TestHurlGrammar_JSONBody(t *testing.T) {
	input := `POST https://example.org/api
Content-Type: application/json
{
    "id": 0,
    "name": "Frieda",
    "age": 3,
    "breed": "Scottish Terrier"
}
HTTP 201`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if req.Body == nil {
		t.Fatal("expected body")
	}
	if req.Body.Type != ast.BodyJSON {
		t.Errorf("expected BodyJSON, got %d", req.Body.Type)
	}
}

func TestHurlGrammar_XMLBody(t *testing.T) {
	input := `GET https://example.org/api/catalog
HTTP 200
<?xml version="1.0" encoding="UTF-8"?>
<catalog>
   <book id="bk101">
      <author>Gambardella, Matthew</author>
   </book>
</catalog>`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	if resp.Body.Type != ast.BodyXML {
		t.Errorf("expected BodyXML, got %d", resp.Body.Type)
	}
}

func TestHurlGrammar_MultilineBody(t *testing.T) {
	input := "GET https://example.org\nHTTP 200\n```\nYear,Make,Model\n1997,Ford,E350\n```"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	if resp.Body.Type != ast.BodyMultiline {
		t.Errorf("expected BodyMultiline, got %d", resp.Body.Type)
	}
	expected := "Year,Make,Model\n1997,Ford,E350"
	if resp.Body.Content != expected {
		t.Errorf("expected %q, got %q", expected, resp.Body.Content)
	}
}

func TestHurlGrammar_MultilineJSONBody(t *testing.T) {
	input := "POST https://example.org\n```json\n{\"key\": \"value\"}\n```\nHTTP 200"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if req.Body == nil {
		t.Fatal("expected body")
	}
	if req.Body.Type != ast.BodyJSON {
		t.Errorf("expected BodyJSON, got %d", req.Body.Type)
	}
}

func TestHurlGrammar_OnelineBody(t *testing.T) {
	input := "POST https://example.org\n`Hello world!`\nHTTP 200"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if req.Body == nil {
		t.Fatal("expected body")
	}
	if req.Body.Type != ast.BodyOneline {
		t.Errorf("expected BodyOneline, got %d", req.Body.Type)
	}
	if req.Body.Content != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", req.Body.Content)
	}
}

func TestHurlGrammar_Base64Body(t *testing.T) {
	input := "GET https://example.org\nHTTP 200\nbase64,TG9yZW0gaXBzdW0=;"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	if resp.Body.Type != ast.BodyBase64 {
		t.Errorf("expected BodyBase64, got %d", resp.Body.Type)
	}
}

func TestHurlGrammar_HexBody(t *testing.T) {
	input := "GET https://example.org\nHTTP 200\nhex,48656c6c6f;"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	if resp.Body.Type != ast.BodyHex {
		t.Errorf("expected BodyHex, got %d", resp.Body.Type)
	}
}

func TestHurlGrammar_FileBody(t *testing.T) {
	input := "GET https://example.org\nHTTP 200\nfile,data.bin;"
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected body")
	}
	if resp.Body.Type != ast.BodyFile {
		t.Errorf("expected BodyFile, got %d", resp.Body.Type)
	}
	if resp.Body.Content != "data.bin" {
		t.Errorf("expected data.bin, got %s", resp.Body.Content)
	}
}

func TestHurlGrammar_Comments(t *testing.T) {
	input := `# A very simple Hurl file
# with tasty comments...
GET https://www.sample.net
x-app: MY_APP  # Inline comment
HTTP 302       # Check redirection`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Comments) < 2 {
		t.Errorf("expected at least 2 comments, got %d", len(file.Comments))
	}
}

func TestHurlGrammar_MultipleEntries(t *testing.T) {
	input := `GET https://example.org/api/token
HTTP 200
[Captures]
token: jsonpath "$.token"

GET https://example.org/api/user
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.name" == "Admin"`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(file.Entries))
	}
	if file.Entries[0].Request.Method != "GET" {
		t.Errorf("expected GET, got %s", file.Entries[0].Request.Method)
	}
	if file.Entries[0].Response.Captures[0].Variable != "token" {
		t.Errorf("expected token capture")
	}
	if file.Entries[1].Request.Headers[0].Value != "Bearer {{token}}" {
		t.Errorf("expected Bearer {{token}} template")
	}
}

func TestHurlGrammar_AssertWithRegexLiteral(t *testing.T) {
	input := `GET https://example.org/hello
HTTP 200
[Asserts]
jsonpath "$.date" matches /^\d{4}-\d{2}-\d{2}$/
jsonpath "$.name" matches /Hello [a-zA-Z]+!/`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 2 {
		t.Fatalf("expected 2 asserts, got %d", len(asserts))
	}
	if asserts[0].Predicate != ast.PredMatches {
		t.Errorf("expected PredMatches, got %d", asserts[0].Predicate)
	}
}

func TestHurlGrammar_AssertPredicateValues(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		valType ast.AssertValueType
	}{
		{"string", `jsonpath "$.x" == "hello"`, ast.ValueString},
		{"int", `jsonpath "$.x" == 42`, ast.ValueInt},
		{"float", `jsonpath "$.x" == 3.14`, ast.ValueFloat},
		{"bool_true", `jsonpath "$.x" == true`, ast.ValueBool},
		{"bool_false", `jsonpath "$.x" == false`, ast.ValueBool},
		{"null", `jsonpath "$.x" == null`, ast.ValueNull},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "GET https://example.org\nHTTP 200\n[Asserts]\n" + tt.input
			p := NewParser(input, "test.hurl")
			file, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			asserts := file.Entries[0].Response.Asserts
			if len(asserts) != 1 {
				t.Fatalf("expected 1 assert, got %d", len(asserts))
			}
			if asserts[0].Value.Type != tt.valType {
				t.Errorf("expected value type %d, got %d", tt.valType, asserts[0].Value.Type)
			}
		})
	}
}

func TestHurlGrammar_RegexQuery(t *testing.T) {
	input := `GET https://example.org/hello
HTTP 200
[Asserts]
regex "^(\\d{4}-\\d{2}-\\d{2})$" == "2018-12-31"
regex /^(\d{4}-\d{2}-\d{2})$/ == "2018-12-31"`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 2 {
		t.Fatalf("expected 2 asserts, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QueryRegex {
		t.Errorf("expected QueryRegex, got %d", asserts[0].Query.Type)
	}
	if asserts[1].Query.Type != ast.QueryRegex {
		t.Errorf("expected QueryRegex, got %d", asserts[1].Query.Type)
	}
}

func TestHurlGrammar_AllFilterTypes(t *testing.T) {
	tests := []struct {
		name      string
		filterStr string
		ftype     ast.FilterType
		fvalue    string
	}{
		{"count", "count", ast.FilterCount, ""},
		{"first", "first", ast.FilterFirst, ""},
		{"last", "last", ast.FilterLast, ""},
		{"nth", "nth 2", ast.FilterNth, "2"},
		{"toInt", "toInt", ast.FilterToInt, ""},
		{"toFloat", "toFloat", ast.FilterToFloat, ""},
		{"toString", "toString", ast.FilterToString, ""},
		{"split", `split ","`, ast.FilterSplit, ","},
		{"regex", `regex "Mr (.*)"`, ast.FilterRegex, "Mr (.*)"},
		{"replace", `replace "old" "new"`, ast.FilterReplace, "old new"},
		{"base64Decode", "base64Decode", ast.FilterBase64Decode, ""},
		{"base64Encode", "base64Encode", ast.FilterBase64Encode, ""},
		{"urlEncode", "urlEncode", ast.FilterUrlEncode, ""},
		{"urlDecode", "urlDecode", ast.FilterUrlDecode, ""},
		{"toHex", "toHex", ast.FilterToHex, ""},
		{"htmlEscape", "htmlEscape", ast.FilterHtmlEscape, ""},
		{"htmlUnescape", "htmlUnescape", ast.FilterHtmlUnescape, ""},
		{"count", "count", ast.FilterCount, ""},
		{"decode", `decode "utf-8"`, ast.FilterDecode, "utf-8"},
		{"toDate", `toDate "%Y-%m-%d"`, ast.FilterToDate, "%Y-%m-%d"},
		{"dateFormat", `dateFormat "%+"`, ast.FilterDateFormat, "%+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "GET https://example.org\nHTTP 200\n[Captures]\nval: jsonpath \"$\" " + tt.filterStr
			p := NewParser(input, "test.hurl")
			file, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error for %s: %v", tt.name, err)
			}
			captures := file.Entries[0].Response.Captures
			if len(captures) != 1 {
				t.Fatalf("expected 1 capture, got %d", len(captures))
			}
			if len(captures[0].Filters) != 1 {
				t.Fatalf("%s: expected 1 filter, got %d", tt.name, len(captures[0].Filters))
			}
			if captures[0].Filters[0].Type != tt.ftype {
				t.Errorf("%s: expected filter type %d, got %d", tt.name, tt.ftype, captures[0].Filters[0].Type)
			}
			if tt.fvalue != "" && captures[0].Filters[0].Value != tt.fvalue {
				t.Errorf("%s: expected value %q, got %q", tt.name, tt.fvalue, captures[0].Filters[0].Value)
			}
		})
	}
}

func TestHurlGrammar_OptionsVariable(t *testing.T) {
	input := `GET https://example.org
[Options]
variable: name=value
variable: count=42
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	opts := file.Entries[0].Request.Options
	if opts.Variables["name"] != "value" {
		t.Errorf("expected name=value, got %s", opts.Variables["name"])
	}
	if opts.Variables["count"] != "42" {
		t.Errorf("expected count=42, got %s", opts.Variables["count"])
	}
}

func TestHurlGrammar_RedirectAssertion(t *testing.T) {
	input := `GET https://example.org/redirecting
[Options]
location: true
HTTP 200
[Asserts]
url == "https://example.org/redirected"
redirects count == 3`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 2 {
		t.Fatalf("expected 2 asserts, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QueryURL {
		t.Errorf("expected QueryURL, got %d", asserts[0].Query.Type)
	}
	if asserts[1].Query.Type != ast.QueryRedirects {
		t.Errorf("expected QueryRedirects, got %d", asserts[1].Query.Type)
	}
}

func TestHurlGrammar_CookieAttributeQuery(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Asserts]
cookie "LSID" == "DQAAAKEaem_vYg"
cookie "LSID[Value]" == "DQAAAKEaem_vYg"
cookie "LSID[Expires]" exists
cookie "LSID[Path]" == "/accounts"
cookie "LSID[Secure]" exists`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 5 {
		t.Fatalf("expected 5 asserts, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QueryCookie {
		t.Errorf("expected QueryCookie, got %d", asserts[0].Query.Type)
	}
	if asserts[0].Query.Value != "LSID" {
		t.Errorf("expected LSID, got %s", asserts[0].Query.Value)
	}
	if asserts[1].Query.Value != "LSID[Value]" {
		t.Errorf("expected LSID[Value], got %s", asserts[1].Query.Value)
	}
}

func TestHurlGrammar_CaptureRedact(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Captures]
secret: header "X-Secret" redact`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	captures := file.Entries[0].Response.Captures
	if len(captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(captures))
	}
	if !captures[0].Redact {
		t.Error("expected redact flag")
	}
}

func TestHurlGrammar_Sha256Assert(t *testing.T) {
	input := `GET https://example.org/data.tar.gz
HTTP 200
[Asserts]
sha256 == hex,039058c6f2c0cb492c533b0a4d14ef77cc0f78abccced5287d84a1a2011cfb81;`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 1 {
		t.Fatalf("expected 1 assert, got %d", len(asserts))
	}
	if asserts[0].Query.Type != ast.QuerySHA256 {
		t.Errorf("expected QuerySHA256, got %d", asserts[0].Query.Type)
	}
}

func TestHurlGrammar_AssertWithHexValue(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Asserts]
bytes contains hex,efbbbf;
bytes startsWith hex,efbbbf;`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if len(asserts) != 2 {
		t.Fatalf("expected 2 asserts, got %d", len(asserts))
	}
}

func TestHurlGrammar_DurationAssert(t *testing.T) {
	input := `GET https://example.org/helloworld
HTTP 200
[Asserts]
duration < 1000`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if asserts[0].Query.Type != ast.QueryDuration {
		t.Errorf("expected QueryDuration, got %d", asserts[0].Query.Type)
	}
	if asserts[0].Value.Int != 1000 {
		t.Errorf("expected 1000, got %d", asserts[0].Value.Int)
	}
}

func TestHurlGrammar_IPQuery(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Asserts]
ip isIpv4
ip not isIpv6`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if asserts[0].Query.Type != ast.QueryIP {
		t.Errorf("expected QueryIP, got %d", asserts[0].Query.Type)
	}
}

func TestHurlGrammar_CertificateQuery(t *testing.T) {
	input := `GET https://example.org
HTTP 200
[Asserts]
certificate "Subject" == "CN=example.org"`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	asserts := file.Entries[0].Response.Asserts
	if asserts[0].Query.Type != ast.QueryCertificate {
		t.Errorf("expected QueryCertificate, got %d", asserts[0].Query.Type)
	}
}

func TestHurlGrammar_ImportExport(t *testing.T) {
	input := `import "common/config.hurlx"
export base_url = "https://api.example.com"

GET {{base_url}}/users
HTTP 200`
	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(file.Imports))
	}
	if file.Imports[0].Path != "common/config.hurlx" {
		t.Errorf("expected common/config.hurlx, got %s", file.Imports[0].Path)
	}
	if len(file.Exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(file.Exports))
	}
	if file.Exports[0].Name != "base_url" {
		t.Errorf("expected base_url, got %s", file.Exports[0].Name)
	}
	if file.Exports[0].Value != "https://api.example.com" {
		t.Errorf("expected https://api.example.com, got %s", file.Exports[0].Value)
	}
}

func TestHurlGrammar_ImportWithAlias(t *testing.T) {
	input := `import "modules/auth.hurlx" as auth
GET https://example.org/api
Authorization: Bearer {{auth.token}}
HTTP 200`
	p := NewParser(input, "test.hurlx")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(file.Imports))
	}
	if file.Imports[0].Alias != "auth" {
		t.Errorf("expected alias auth, got %s", file.Imports[0].Alias)
	}
}

func TestHurlGrammar_SectionAliases(t *testing.T) {
	input := `GET https://example.org
[QueryStringParams]
page: 1
[FormParams]
name: test
[MultipartFormData]
file1: file,data.bin;
HTTP 200`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	req := file.Entries[0].Request
	if len(req.Query) != 1 {
		t.Fatalf("expected 1 query param, got %d", len(req.Query))
	}
	if len(req.Form) != 1 {
		t.Fatalf("expected 1 form param, got %d", len(req.Form))
	}
	if len(req.Multipart) != 1 {
		t.Fatalf("expected 1 multipart field, got %d", len(req.Multipart))
	}
}

func TestHurlGrammar_AllHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			input := m + " https://example.org"
			p := NewParser(input, "test.hurl")
			file, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if file.Entries[0].Request.Method != m {
				t.Errorf("expected %s, got %s", m, file.Entries[0].Request.Method)
			}
		})
	}
}

func TestHurlGrammar_ImplicitBodyAssert(t *testing.T) {
	input := `GET https://example.org/api/dogs/1
HTTP 200
{
    "id": 1,
    "name": "Frieda"
}`
	p := NewParser(input, "test.hurl")
	file, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	resp := file.Entries[0].Response
	if resp.Body == nil {
		t.Fatal("expected response body")
	}
	if resp.Body.Type != ast.BodyJSON {
		t.Errorf("expected BodyJSON, got %d", resp.Body.Type)
	}
}
