package runner

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wei-lli/hurlx/ast"
	"github.com/wei-lli/hurlx/tmpl"
)

func TestIsString(t *testing.T) {
	if !isString("hello") {
		t.Error("expected string to be string")
	}
	if isString(42) {
		t.Error("expected int not to be string")
	}
	if isString(nil) {
		t.Error("expected nil not to be string")
	}
}

func TestIsInteger(t *testing.T) {
	if !isInteger(42) {
		t.Error("expected int to be integer")
	}
	if !isInteger(int64(42)) {
		t.Error("expected int64 to be integer")
	}
	if !isInteger(int32(42)) {
		t.Error("expected int32 to be integer")
	}
	if isInteger(42.0) {
		t.Error("expected float64 not to be integer")
	}
	if isInteger("42") {
		t.Error("expected string not to be integer")
	}
}

func TestIsFloat(t *testing.T) {
	if !isFloat(3.14) {
		t.Error("expected float64 to be float")
	}
	if !isFloat(float32(3.14)) {
		t.Error("expected float32 to be float")
	}
	if isFloat(42) {
		t.Error("expected int not to be float")
	}
}

func TestIsBool(t *testing.T) {
	if !isBool(true) {
		t.Error("expected true to be bool")
	}
	if !isBool(false) {
		t.Error("expected false to be bool")
	}
	if isBool("true") {
		t.Error("expected string not to be bool")
	}
}

func TestIsList(t *testing.T) {
	if !isList([]interface{}{1, 2}) {
		t.Error("expected []interface{} to be list")
	}
	if isList("hello") {
		t.Error("expected string not to be list")
	}
}

func TestIsObject(t *testing.T) {
	if !isObject(map[string]interface{}{"a": 1}) {
		t.Error("expected map to be object")
	}
	if isObject("hello") {
		t.Error("expected string not to be object")
	}
}

func TestIsEmpty(t *testing.T) {
	if !isEmpty("") {
		t.Error("expected empty string to be empty")
	}
	if !isEmpty([]interface{}{}) {
		t.Error("expected empty slice to be empty")
	}
	if !isEmpty(map[string]interface{}{}) {
		t.Error("expected empty map to be empty")
	}
	if isEmpty("hello") {
		t.Error("expected non-empty string not to be empty")
	}
	if isEmpty(42) {
		t.Error("expected int not to be empty")
	}
}

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		input interface{}
		want  bool
	}{
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"1.2.3", false},
		{"999.999.999.999", false},
		{"192.168.1.256", false},
		{"hello", false},
		{42, false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isIPv4(tt.input); got != tt.want {
			t.Errorf("isIPv4(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		input interface{}
		want  bool
	}{
		{"::1", true},
		{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"fe80::1", true},
		{"2001:db8::1", true},
		{"[::1]", true},
		{"192.168.1.1", false},
		{"hello:world", false},
		{"not an ip", false},
		{42, false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isIPv6(tt.input); got != tt.want {
			t.Errorf("isIPv6(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input interface{}
		want  bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"550e8400-e29b-41d4-a716", false},
		{"550e8400-e29b-41d4-a716-446655440000-extra", false},
		{"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", false},
		{"550e8400-e29b_41d4-a716-446655440000", false},
		{"--------|----|----|----|------------", false},
		{42, false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isUUID(tt.input); got != tt.want {
			t.Errorf("isUUID(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsNumber(t *testing.T) {
	if !isNumber(42) {
		t.Error("expected int to be number")
	}
	if !isNumber(3.14) {
		t.Error("expected float64 to be number")
	}
	if isNumber("42") {
		t.Error("expected string not to be number")
	}
}

func TestIsISODate(t *testing.T) {
	if !isISODate("2024-01-15T10:30:00Z") {
		t.Error("expected valid ISO date")
	}
	if !isISODate("2024-01-15T10:30:00+08:00") {
		t.Error("expected valid ISO date with timezone")
	}
	if isISODate("not-a-date") {
		t.Error("expected invalid ISO date")
	}
}

func TestIsCollection(t *testing.T) {
	if !isCollection([]interface{}{1, 2}) {
		t.Error("expected slice to be collection")
	}
	if !isCollection(map[string]interface{}{"a": 1}) {
		t.Error("expected map to be collection")
	}
	if isCollection("hello") {
		t.Error("expected string not to be collection")
	}
}

func TestIsDate(t *testing.T) {
	if !isDate("2024-01-15T10:30:00Z") {
		t.Error("expected RFC3339 date")
	}
	if !isDate("2024-01-15") {
		t.Error("expected date-only format")
	}
	if isDate("not-a-date") {
		t.Error("expected invalid date")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"100ms", 100 * time.Millisecond},
		{"5s", 5 * time.Second},
		{"2m", 2 * time.Minute},
		{"1h", 1 * time.Hour},
		{"500", 500 * time.Millisecond},
		{"invalid", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := ParseDuration(tt.input); got != tt.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		actual   interface{}
		expected ast.AssertValue
		want     int // 0 = equal, -1 = less, 1 = greater
	}{
		{"hello", ast.AssertValue{Type: ast.ValueString, Str: "hello"}, 0},
		{"abc", ast.AssertValue{Type: ast.ValueString, Str: "def"}, -1},
		{42, ast.AssertValue{Type: ast.ValueInt, Int: 42}, 0},
		{int64(42), ast.AssertValue{Type: ast.ValueInt, Int: 42}, 0},
		{10, ast.AssertValue{Type: ast.ValueInt, Int: 20}, -1},
		{3.14, ast.AssertValue{Type: ast.ValueFloat, Float: 3.14}, 0},
		{true, ast.AssertValue{Type: ast.ValueBool, Bool: true}, 0},
		{nil, ast.AssertValue{Type: ast.ValueNull}, 0},
	}
	for _, tt := range tests {
		if got := compareValues(tt.actual, tt.expected); got != tt.want {
			t.Errorf("compareValues(%v, %v) = %d, want %d", tt.actual, tt.expected, got, tt.want)
		}
	}
}

func TestCheckContains(t *testing.T) {
	if err := checkContains("hello world", ast.AssertValue{Type: ast.ValueString, Str: "world"}, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkContains("hello", ast.AssertValue{Type: ast.ValueString, Str: "world"}, false); err == nil {
		t.Error("expected error for missing substring")
	}
	if err := checkContains("hello world", ast.AssertValue{Type: ast.ValueString, Str: "world"}, true); err == nil {
		t.Error("expected error for 'not contains' when substring present")
	}
}

func TestCheckStartsWith(t *testing.T) {
	if err := checkStartsWith("hello world", ast.AssertValue{Type: ast.ValueString, Str: "hello"}, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkStartsWith("hello", ast.AssertValue{Type: ast.ValueString, Str: "world"}, false); err == nil {
		t.Error("expected error for non-matching prefix")
	}
}

func TestCheckEndsWith(t *testing.T) {
	if err := checkEndsWith("hello world", ast.AssertValue{Type: ast.ValueString, Str: "world"}, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkEndsWith("hello", ast.AssertValue{Type: ast.ValueString, Str: "world"}, false); err == nil {
		t.Error("expected error for non-matching suffix")
	}
}

func TestCheckMatches(t *testing.T) {
	if err := checkMatches("hello123", ast.AssertValue{Type: ast.ValueString, Str: "hello\\d+"}, false); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkMatches("hello", ast.AssertValue{Type: ast.ValueString, Str: "^\\d+$"}, false); err == nil {
		t.Error("expected error for non-matching regex")
	}
}

func TestCheckMatchesTooLong(t *testing.T) {
	longPattern := make([]byte, maxRegexPatternLen+1)
	for i := range longPattern {
		longPattern[i] = 'a'
	}
	err := checkMatches("test", ast.AssertValue{Type: ast.ValueString, Str: string(longPattern)}, false)
	if err == nil {
		t.Error("expected error for too-long regex pattern")
	}
}

func TestOptsFromEntry(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 0},
		{5, 4},
		{-1, 0},
	}
	for _, tt := range tests {
		if got := optsFromEntry(tt.input); got != tt.want {
			t.Errorf("optsFromEntry(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestOptsToEntry(t *testing.T) {
	tests := []struct {
		n     int
		total int
		want  int
	}{
		{0, 10, 10},
		{5, 10, 5},
		{15, 10, 10},
		{-1, 10, 10},
	}
	for _, tt := range tests {
		if got := optsToEntry(tt.n, tt.total); got != tt.want {
			t.Errorf("optsToEntry(%d, %d) = %d, want %d", tt.n, tt.total, got, tt.want)
		}
	}
}

func TestResolveFilePath(t *testing.T) {
	if got := resolveFilePath("/root", "file.txt"); got != "/root/file.txt" {
		t.Errorf("resolveFilePath(/root, file.txt) = %q", got)
	}
	if got := resolveFilePath("", "file.txt"); got != "file.txt" {
		t.Errorf("resolveFilePath('', file.txt) = %q", got)
	}
	if got := resolveFilePath("/root", "/abs/path.txt"); got != "/abs/path.txt" {
		t.Errorf("resolveFilePath(/root, /abs/path.txt) = %q", got)
	}
}

func TestBuildBody(t *testing.T) {
	r := NewRunner(RunOptions{})
	tests := []struct {
		name  string
		body  *ast.Body
		check func([]byte) bool
	}{
		{
			"json body",
			&ast.Body{Type: ast.BodyJSON, Content: `{"key":"value"}`},
			func(b []byte) bool { return string(b) == `{"key":"value"}` },
		},
		{
			"xml body",
			&ast.Body{Type: ast.BodyXML, Content: "<root/>"},
			func(b []byte) bool { return string(b) == "<root/>" },
		},
		{
			"hex body",
			&ast.Body{Type: ast.BodyHex, Content: "48656c6c6f"},
			func(b []byte) bool { return string(b) == "Hello" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.buildBody(tt.body)
			if !tt.check(got) {
				t.Errorf("buildBody(%s) = %q", tt.name, got)
			}
		})
	}
}

func TestRunSimpleGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	entries := []ast.Entry{
		{
			Request: &ast.Request{
				Method: "GET",
				URL:    server.URL + "/test",
			},
			Response: &ast.Response{
				Status: 200,
			},
		},
	}

	r := NewRunner(RunOptions{})
	result, err := r.Run(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if string(result.Entries[0].Body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %q", string(result.Entries[0].Body))
	}
}

func TestRunWithCapture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"abc123"}`))
	}))
	defer server.Close()

	entries := []ast.Entry{
		{
			Request: &ast.Request{
				Method: "GET",
				URL:    server.URL + "/auth",
			},
			Response: &ast.Response{
				Status: 200,
				Captures: []ast.Capture{
					{
						Variable: "token",
						Query:    ast.Query{Type: ast.QueryJSONPath, Value: "$.token"},
					},
				},
			},
		},
	}

	r := NewRunner(RunOptions{})
	result, err := r.Run(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Entries[0].Captures["token"] != "abc123" {
		t.Errorf("expected token=abc123, got %v", result.Entries[0].Captures["token"])
	}
}

func TestRunSSRFBlocked(t *testing.T) {
	entries := []ast.Entry{
		{
			Request: &ast.Request{
				Method: "GET",
				URL:    "file:///etc/passwd",
			},
		},
	}

	r := NewRunner(RunOptions{})
	_, err := r.Run(entries)
	if err == nil {
		t.Error("expected error for file:// URL scheme")
	}
}

func TestRunWithVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"name":"test"}`))
	}))
	defer server.Close()

	vars := tmpl.NewVariables()
	vars.Set("base_url", server.URL)

	entries := []ast.Entry{
		{
			Request: &ast.Request{
				Method: "GET",
				URL:    "{{base_url}}/api",
			},
			Response: &ast.Response{
				Status: 200,
			},
		},
	}

	r := NewRunner(RunOptions{Variables: vars})
	result, err := r.Run(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}
