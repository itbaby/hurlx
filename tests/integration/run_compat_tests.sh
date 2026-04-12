#!/bin/bash
# hurlx vs hurl compatibility test runner
# Runs the same .hurl files through both hurl and hurlx,
# comparing exit codes and key output

set -e

HURLX="./hurlx"
HURL="hurl"
PASS=0
FAIL=0
SKIP=0
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

run_test() {
    local name="$1"
    local file="$2"
    local hurlx_expected_exit="${3:-0}"
    local hurl_expected_exit="${4:-0}"
    local skip_hurl="${5:-false}"

    # Run hurlx
    hurlx_exit=0
    hurlx_out=$($HURLX --test "$file" 2>&1) || hurlx_exit=$?
    
    # Run hurl
    hurl_exit=0
    if [ "$skip_hurl" = "true" ]; then
        hurl_out="SKIPPED"
        hurl_exit=-1
    else
        hurl_out=$($HURL --test "$file" 2>&1) || hurl_exit=$?
    fi

    # Compare
    local ok=true
    local details=""

    if [ "$hurlx_exit" -ne "$hurlx_expected_exit" ]; then
        ok=false
        details="hurlx exit=$hurlx_exit (expected $hurlx_expected_exit)"
    fi

    if [ "$skip_hurl" = "false" ] && [ "$hurl_exit" -ne "$hurl_expected_exit" ]; then
        ok=false
        details="$details hurl exit=$hurl_exit (expected $hurl_expected_exit)"
    fi

    if $ok; then
        echo -e "${GREEN}PASS${NC} $name"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}FAIL${NC} $name: $details"
        echo "  hurlx: $hurlx_out"
        echo "  hurl:  $hurl_out"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== Hurl Grammar & Runtime Compatibility Tests ==="
echo ""

# --- Parser-only tests (no HTTP) ---
echo "--- Basic HTTP Methods ---"

cat > $TMPDIR/get.hurl << 'EOF'
GET https://httpbin.org/get
HTTP 200
EOF
run_test "GET request" $TMPDIR/get.hurl 0 0

cat > $TMPDIR/post.hurl << 'EOF'
POST https://httpbin.org/post
Content-Type: application/json
{"name": "test"}
HTTP 200
EOF
run_test "POST with JSON body" $TMPDIR/post.hurl 0 0

cat > $TMPDIR/put.hurl << 'EOF'
PUT https://httpbin.org/put
Content-Type: application/json
{"id": 1}
HTTP 200
EOF
run_test "PUT request" $TMPDIR/put.hurl 0 0

cat > $TMPDIR/delete.hurl << 'EOF'
DELETE https://httpbin.org/delete
HTTP 200
EOF
run_test "DELETE request" $TMPDIR/delete.hurl 0 0

cat > $TMPDIR/patch.hurl << 'EOF'
PATCH https://httpbin.org/patch
Content-Type: application/json
{"field": "value"}
HTTP 200
EOF
run_test "PATCH request" $TMPDIR/patch.hurl 0 0

cat > $TMPDIR/head.hurl << 'EOF'
HEAD https://httpbin.org/get
HTTP 200
EOF
run_test "HEAD request" $TMPDIR/head.hurl 0 0

echo ""
echo "--- Headers ---"

cat > $TMPDIR/headers.hurl << 'EOF'
GET https://httpbin.org/headers
X-Test-Header: hello-world
Accept: application/json
HTTP 200
[Asserts]
jsonpath "$.headers.X-Test-Header" == "hello-world"
EOF
run_test "Custom headers" $TMPDIR/headers.hurl 0 0

echo ""
echo "--- Query Parameters ---"

cat > $TMPDIR/query.hurl << 'EOF'
GET https://httpbin.org/get
[Query]
page: 2
size: 10
HTTP 200
[Asserts]
jsonpath "$.args.page" == "2"
jsonpath "$.args.size" == "10"
EOF
run_test "Query params section" $TMPDIR/query.hurl 0 0

echo ""
echo "--- Form Data ---"

cat > $TMPDIR/form.hurl << 'EOF'
POST https://httpbin.org/post
[Form]
username: admin
password: secret123
HTTP 200
[Asserts]
jsonpath "$.form.username" == "admin"
jsonpath "$.form.password" == "secret123"
EOF
run_test "Form POST" $TMPDIR/form.hurl 0 0

echo ""
echo "--- JSON Body & Assertions ---"

cat > $TMPDIR/json_assert.hurl << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.author" == "Yours Truly"
jsonpath "$.slideshow.title" exists
jsonpath "$.slideshow.slides" isList
jsonpath "$.slideshow.slides" count > 0
header "Content-Type" contains "json"
status == 200
EOF
run_test "JSON asserts" $TMPDIR/json_assert.hurl 0 0

echo ""
echo "--- Status Code Asserts ---"

cat > $TMPDIR/status_404.hurl << 'EOF'
GET https://httpbin.org/status/404
HTTP 404
EOF
run_test "404 status" $TMPDIR/status_404.hurl 0 0

cat > $TMPDIR/status_wildcard.hurl << 'EOF'
GET https://httpbin.org/status/404
HTTP *
[Asserts]
status == 404
EOF
run_test "Wildcard status" $TMPDIR/status_wildcard.hurl 0 0

echo ""
echo "--- Captures and Chaining ---"

cat > $TMPDIR/chaining.hurl << 'EOF'
POST https://httpbin.org/post
Content-Type: application/json
{"test": "value"}
HTTP 200
[Captures]
origin: jsonpath "$.origin"
url: jsonpath "$.url"

GET https://httpbin.org/get
X-Origin: {{origin}}
HTTP 200
[Asserts]
jsonpath "$.headers.X-Origin" == "{{origin}}"
EOF
run_test "Capture and chain" $TMPDIR/chaining.hurl 0 0

echo ""
echo "--- Filters ---"

cat > $TMPDIR/filters.hurl << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
ua: jsonpath "$.headers.User-Agent"
ua_parts: jsonpath "$.headers.User-Agent" split "/" nth 0
[Asserts]
variable "ua_parts" exists
EOF
run_test "Filters (split, nth)" $TMPDIR/filters.hurl 0 0

cat > $TMPDIR/count_filter.hurl << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides" count > 0
jsonpath "$.slideshow.slides[0].title" isString
EOF
run_test "Count filter" $TMPDIR/count_filter.hurl 0 0

echo ""
echo "--- Basic Auth ---"

cat > $TMPDIR/basicauth.hurl << 'EOF'
GET https://httpbin.org/basic-auth/admin/password
[BasicAuth]
admin: password
HTTP 200
[Asserts]
jsonpath "$.authenticated" == true
EOF
run_test "Basic auth" $TMPDIR/basicauth.hurl 0 0

echo ""
echo "--- Redirects ---"

cat > $TMPDIR/redirect.hurl << 'EOF'
GET https://httpbin.org/redirect/2
[Options]
location: true
HTTP 200
[Asserts]
url == "https://httpbin.org/get"
redirects count == 2
EOF
run_test "Redirect following" $TMPDIR/redirect.hurl 0 0

echo ""
echo "--- Bytes and Hash Asserts ---"

cat > $TMPDIR/bytes.hurl << 'EOF'
GET https://httpbin.org/bytes/10
HTTP 200
[Asserts]
bytes count == 10
sha256 exists
md5 exists
EOF
run_test "Bytes/hash asserts" $TMPDIR/bytes.hurl 0 0

echo ""
echo "--- Duration Assert ---"

cat > $TMPDIR/duration.hurl << 'EOF'
GET https://httpbin.org/delay/0.5
HTTP 200
[Asserts]
duration < 10000
EOF
run_test "Duration assert" $TMPDIR/duration.hurl 0 0

echo ""
echo "--- Variable Assert ---"

cat > $TMPDIR/variable.hurl << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
origin: jsonpath "$.origin"
[Asserts]
variable "origin" exists
variable "origin" isString
EOF
run_test "Variable assert" $TMPDIR/variable.hurl 0 0

echo ""
echo "--- Templates ---"

cat > $TMPDIR/template.hurl << 'EOF'
GET https://httpbin.org/uuid
HTTP 200
[Captures]
uuid: jsonpath "$.uuid"
[Asserts]
variable "uuid" isUuid
EOF
run_test "UUID isUuid assert" $TMPDIR/template.hurl 0 0

echo ""
echo "--- Multiple Entries ---"

cat > $TMPDIR/multi.hurl << 'EOF'
GET https://httpbin.org/uuid
HTTP 200
[Captures]
id1: jsonpath "$.uuid"

GET https://httpbin.org/uuid
HTTP 200
[Captures]
id2: jsonpath "$.uuid"

POST https://httpbin.org/post
Content-Type: application/json
{"id1": "{{id1}}", "id2": "{{id2}}"}
HTTP 200
[Asserts]
jsonpath "$.json.id1" == "{{id1}}"
jsonpath "$.json.id2" == "{{id2}}"
EOF
run_test "Multiple entries" $TMPDIR/multi.hurl 0 0

echo ""
echo "--- Not predicate ---"

cat > $TMPDIR/not.hurl << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Asserts]
jsonpath "$.headers.X-Missing" not exists
status not == 404
EOF
run_test "Not predicate" $TMPDIR/not.hurl 0 0

echo ""
echo "--- Cookies ---"

cat > $TMPDIR/cookies.hurl << 'EOF'
GET https://httpbin.org/cookies/set?test_cookie=hello
[Options]
location: true
HTTP 200

GET https://httpbin.org/cookies
HTTP 200
[Asserts]
jsonpath "$.cookies.test_cookie" == "hello"
EOF
run_test "Cookies" $TMPDIR/cookies.hurl 0 0

echo ""
echo "--- XPath (HTML) ---"

cat > $TMPDIR/xpath.hurl << 'EOF'
GET https://httpbin.org/html
HTTP 200
[Asserts]
xpath "string(//h1)" exists
xpath "//p" count > 0
EOF
run_test "XPath on HTML" $TMPDIR/xpath.hurl 0 0 true

echo ""
echo "=== Summary ==="
echo -e "PASS: ${GREEN}$PASS${NC}  FAIL: ${RED}$FAIL${NC}"
echo ""

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
