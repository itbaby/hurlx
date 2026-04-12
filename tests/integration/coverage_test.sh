#!/bin/bash
# Comprehensive Hurlx Coverage Test
# Tests all features from hurl.dev documentation

OUTPUT_DIR="examples/coverage"
mkdir -p "$OUTPUT_DIR"

PASS=0
FAIL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Hurlx Comprehensive Coverage Test ===${NC}"
echo ""

# Helper function
run_test() {
    local name="$1"
    local file="$2"
    ./hurlx --test "$file" 2>&1 > /dev/null
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $name"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}✗${NC} $name"
        FAIL=$((FAIL + 1))
    fi
}

# ===== SECTION 1: File Format =====
echo -e "${BLUE}--- 1. File Format ---${NC}"

# Comments
cat > "$OUTPUT_DIR/01_comment.hurlx" << 'EOF'
# This is a comment
GET https://httpbin.org/get
# Another comment
HTTP 200
EOF
run_test "Comments" "$OUTPUT_DIR/01_comment.hurlx"

# Special characters in strings
cat > "$OUTPUT_DIR/02_special_chars.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.title" == "Test \u{1F600}"
EOF
run_test "Unicode escape" "$OUTPUT_DIR/02_special_chars.hurlx"

# ===== SECTION 2: Request Methods =====
echo -e "${BLUE}--- 2. Request Methods ---${NC}"

cat > "$OUTPUT_DIR/03_methods.hurlx" << 'EOF'
OPTIONS https://httpbin.org/get
HTTP 200
EOF
run_test "OPTIONS method" "$OUTPUT_DIR/03_methods.hurlx"

# ===== SECTION 3: Request Sections =====
echo -e "${BLUE}--- 3. Request Sections ---${NC}"

# Query section (alias)
cat > "$OUTPUT_DIR/04_query_alias.hurlx" << 'EOF'
GET https://httpbin.org/get
[QueryStringParams]
key: value
HTTP 200
EOF
run_test "[QueryStringParams] section" "$OUTPUT_DIR/04_query_alias.hurlx"

# Form section (alias)
cat > "$OUTPUT_DIR/05_form_alias.hurlx" << 'EOF'
POST https://httpbin.org/post
[FormParams]
name: test
HTTP 200
EOF
run_test "[FormParams] section" "$OUTPUT_DIR/05_form_alias.hurlx"

# Multipart section (alias)
cat > "$OUTPUT_DIR/06_multipart_alias.hurlx" << 'EOF'
POST https://httpbin.org/post
[MultipartFormData]
file1: file,data.bin;
HTTP 200
EOF
run_test "[MultipartFormData] section" "$OUTPUT_DIR/06_multipart_alias.hurlx"

# ===== SECTION 4: Request Body Types =====
echo -e "${BLUE}--- 4. Request Body Types ---${NC}"

# JSON body (inline)
cat > "$OUTPUT_DIR/07_json_body.hurlx" << 'EOF'
POST https://httpbin.org/post
Content-Type: application/json
{"name": "test"}
HTTP 200
EOF
run_test "JSON body (inline)" "$OUTPUT_DIR/07_json_body.hurlx"

# JSON body (multiline)
cat > "$OUTPUT_DIR/08_json_multiline.hurlx" << 'EOF'
POST https://httpbin.org/post
```json
{"name": "test"}
```
HTTP 200
EOF
run_test "JSON body (multiline)" "$OUTPUT_DIR/08_json_multiline.hurlx"

# XML body
cat > "$OUTPUT_DIR/09_xml_body.hurlx" << 'EOF'
GET https://httpbin.org/xml
HTTP 200
<?xml version="1.0"?>
<root/>
EOF
run_test "XML body" "$OUTPUT_DIR/09_xml_body.hurlx"

# Multiline string body
cat > "$OUTPUT_DIR/10_multiline_body.hurlx" << 'EOF'
GET https://httpbin.org/robots.txt
HTTP 200
```
User-agent: *
Disallow: /
```
EOF
run_test "Multiline body" "$OUTPUT_DIR/10_multiline_body.hurlx"

# Oneline string body
cat > "$OUTPUT_DIR/11_oneline_body.hurlx" << 'EOF'
POST https://httpbin.org/post
`hello world`
HTTP 200
EOF
run_test "Oneline body" "$OUTPUT_DIR/11_oneline_body.hurlx"

# ===== SECTION 5: Response =====
echo -e "${BLUE}--- 5. Response ---${NC}"

# Version check
cat > "$OUTPUT_DIR/12_version.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP *
[Asserts]
version matches "1\\.1"
EOF
run_test "Version assert" "$OUTPUT_DIR/12_version.hurlx"

# Implicit header assert
cat > "$OUTPUT_DIR/13_implicit_header.hurlx" << 'EOF'
GET https://httpbin.org/json
Content-Type: application/json
HTTP 200
EOF
run_test "Implicit header assert" "$OUTPUT_DIR/13_implicit_header.hurlx"

# ===== SECTION 6: Queries =====
echo -e "${BLUE}--- 6. Queries ---${NC}"

# Certificate query
cat > "$OUTPUT_DIR/14_certificate.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Asserts]
certificate "Subject" exists
EOF
run_test "Certificate query" "$OUTPUT_DIR/14_certificate.hurlx"

# ===== SECTION 7: Predicates =====
echo -e "${BLUE}--- 7. Predicates ---${NC}"

# isNumber
cat > "$OUTPUT_DIR/15_isnumber.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides[0].type" isString
EOF
run_test "isString predicate" "$OUTPUT_DIR/15_isnumber.hurlx"

# isDate
cat > "$OUTPUT_DIR/16_isdate.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
date_header: header "Date"
[Asserts]
variable "date_header" isIsoDate
EOF
run_test "isIsoDate predicate" "$OUTPUT_DIR/16_isdate.hurlx"

# includes predicate (on collection)
cat > "$OUTPUT_DIR/17_includes.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides[*].type" includes "all"
EOF
run_test "includes predicate" "$OUTPUT_DIR/17_includes.hurlx"

# ===== SECTION 8: Filters =====
echo -e "${BLUE}--- 8. Filters ---${NC}"

# urlQueryParam filter
cat > "$OUTPUT_DIR/18_urlqueryparam.hurlx" << 'EOF'
GET https://httpbin.org/get?foo=bar&baz=qux
HTTP 200
[Asserts]
url urlQueryParam "foo" == "bar"
url urlQueryParam "baz" == "qux"
EOF
run_test "urlQueryParam filter" "$OUTPUT_DIR/18_urlqueryparam.hurlx"

# toHex filter
cat > "$OUTPUT_DIR/19_tohex.hurlx" << 'EOF'
GET https://httpbin.org/bytes/5
HTTP 200
[Asserts]
bytes toHex exists
EOF
run_test "toHex filter" "$OUTPUT_DIR/19_tohex.hurlx"

# toFloat filter
cat > "$OUTPUT_DIR/20_tofloat.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.date" toFloat > 0
EOF
run_test "toFloat filter" "$OUTPUT_DIR/20_tofloat.hurlx"

# htmlEscape filter
cat > "$OUTPUT_DIR/21_html_escape.hurlx" << 'EOF'
GET https://httpbin.org/html
HTTP 200
[Captures]
title: xpath "string(//title)"
[Asserts]
variable "title" htmlEscape exists
EOF
run_test "htmlEscape filter" "$OUTPUT_DIR/21_html_escape.hurlx"

# htmlUnescape filter
cat > "$OUTPUT_DIR/22_html_unescape.hurlx" << 'EOF'
GET https://httpbin.org/html
HTTP 200
[Captures]
title: xpath "string(//title)"
[Asserts]
variable "title" htmlUnescape exists
EOF
run_test "htmlUnescape filter" "$OUTPUT_DIR/22_html_unescape.hurlx"

# base64UrlSafeEncode filter
cat > "$OUTPUT_DIR/23_base64urlsafebl.hurlx" << 'EOF'
GET https://httpbin.org/base64/aGVsbG8gd29ybGQ=
HTTP 200
[Asserts]
body base64UrlSafeDecode contains "hello world"
EOF
run_test "base64UrlSafeDecode filter" "$OUTPUT_DIR/23_base64urlsafebl.hurlx"

# first filter
cat > "$OUTPUT_DIR/24_first.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides" first == "Wake up to WonderWidgets!"
EOF
run_test "first filter" "$OUTPUT_DIR/24_first.hurlx"

# last filter
cat > "$OUTPUT_DIR/25_last.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides" last isString
EOF
run_test "last filter" "$OUTPUT_DIR/25_last.hurlx"

# replaceRegex filter
cat > "$OUTPUT_DIR/26_replaceregex.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
body replaceRegex "World" "Universe" contains "Universe"
EOF
run_test "replaceRegex filter" "$OUTPUT_DIR/26_replaceregex.hurlx"

# toDate filter with format
cat > "$OUTPUT_DIR/27_todate.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
date: header "Date"
[Asserts]
variable "date" toDate "%a, %d %b %Y %H:%M:%S %z" exists
EOF
run_test "toDate filter" "$OUTPUT_DIR/27_todate.hurlx"

# daysAfterNow filter
cat > "$OUTPUT_DIR/28_daysafternow.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
date: header "Date"
[Asserts]
variable "date" toDate "%a, %d %b %Y %H:%M:%S %z" daysAfterNow exists
EOF
run_test "daysAfterNow filter" "$OUTPUT_DIR/28_daysafternow.hurlx"

# location filter
cat > "$OUTPUT_DIR/29_location.hurlx" << 'EOF'
GET https://httpbin.org/redirect-to?url=https://example.com
[Options]
location: true
HTTP 200
[Asserts]
redirects count == 1
redirects nth 0 location == "https://example.com"
EOF
run_test "location filter" "$OUTPUT_DIR/29_location.hurlx"

# ===== SECTION 9: Templates =====
echo -e "${BLUE}--- 9. Templates ---${NC}"

# getEnv function
cat > "$OUTPUT_DIR/30_getenv.hurlx" << 'EOF'
GET https://httpbin.org/anything
User-Agent: {{getEnv "USER"}}
HTTP 200
[Asserts]
jsonpath "$.headers.User-Agent" exists
EOF
run_test "getEnv template" "$OUTPUT_DIR/30_getenv.hurlx"

# ===== SECTION 10: Import/Export (Hurlx specific) =====
echo -e "${BLUE}--- 10. Import/Export (Hurlx) ---${NC}"

# Create common module
mkdir -p "$OUTPUT_DIR/modules"
cat > "$OUTPUT_DIR/modules/config.hurlx" << 'EOF'
export base_url = "https://httpbin.org"
export api_version = "v1"
EOF

# Import module
cat > "$OUTPUT_DIR/31_import.hurlx" << 'EOF'
import "modules/config.hurlx"

GET {{base_url}}/get
HTTP 200
[Asserts]
header "Content-Type" contains "json"
EOF
run_test "Import module" "$OUTPUT_DIR/31_import.hurlx"

# ===== SECTION 11: Advanced Features =====
echo -e "${BLUE}--- 11. Advanced Features ---${NC}"

# Multiple asserts
cat > "$OUTPUT_DIR/32_multiple_asserts.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
status == 200
header "Content-Type" contains "json"
jsonpath "$.slideshow.author" == "Yours Truly"
jsonpath "$.slideshow.slides" isList
jsonpath "$.slideshow.slides" count > 0
jsonpath "$.slideshow.date" != null
body contains "slideshow"
bytes count > 0
duration < 10000
EOF
run_test "Multiple asserts" "$OUTPUT_DIR/32_multiple_asserts.hurlx"

# Negative predicates
cat > "$OUTPUT_DIR/33_not_predicate.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Asserts]
header "X-Nonexistent" not exists
status not == 500
jsonpath "$.headers.X-Missing" not exists
EOF
run_test "Not predicates" "$OUTPUT_DIR/33_not_predicate.hurlx"

# Chained filters
cat > "$OUTPUT_DIR/34_chained_filters.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Captures]
title: jsonpath "$.slideshow.title" toString upper
[Asserts]
variable "title" == "SAMPLE SLIDE SHOW"
EOF
run_test "Chained filters (toString + more)" "$OUTPUT_DIR/34_chained_filters.hurlx"

# Capture with regex filter
cat > "$OUTPUT_DIR/35_capture_regex.hurlx" << 'EOF'
GET https://httpbin.org/headers
HTTP 200
[Captures]
content_type: header "Content-Type" regex "([\\w/]+)"
[Asserts]
variable "content_type" == "application/json"
EOF
run_test "Capture with regex filter" "$OUTPUT_DIR/35_capture_regex.hurlx"

# ===== SECTION 12: Edge Cases =====
echo -e "${BLUE}--- 12. Edge Cases ---${NC}"

# Empty body
cat > "$OUTPUT_DIR/36_empty.hurlx" << 'EOF'
GET https://httpbin.org/bytes/0
HTTP 200
[Asserts]
bytes count == 0
EOF
run_test "Empty body" "$OUTPUT_DIR/36_empty.hurlx"

# Large number
cat > "$OUTPUT_DIR/37_large_number.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides[0].type" != 0
EOF
run_test "String vs number" "$OUTPUT_DIR/37_large_number.hurlx"

# Null value
cat > "$OUTPUT_DIR/38_null_value.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.date" != null
jsonpath "$.slideshow.date" isString
EOF
run_test "Null value" "$OUTPUT_DIR/38_null_value.hurlx"

# ===== SUMMARY =====
echo ""
echo -e "${BLUE}=== Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ All coverage tests passed!${NC}"
else
    echo -e "${RED}✗ Some tests failed. Review above.${NC}"
fi