#!/bin/bash
# 生成所有测试用例的示例文件

OUTPUT_DIR="examples/tests"

mkdir -p "$OUTPUT_DIR"

echo "=== 生成 Hurlx 测试用例示例 ==="
echo ""

# 1. Basic HTTP Methods
echo "1. GET request"
cat > "$OUTPUT_DIR/01_get.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
EOF

echo "2. POST with JSON body"
cat > "$OUTPUT_DIR/02_post_json.hurlx" << 'EOF'
POST https://httpbin.org/post
Content-Type: application/json
{"name": "test"}
HTTP 200
EOF

echo "3. PUT request"
cat > "$OUTPUT_DIR/03_put.hurlx" << 'EOF'
PUT https://httpbin.org/put
Content-Type: application/json
{"id": 1}
HTTP 200
EOF

echo "4. DELETE request"
cat > "$OUTPUT_DIR/04_delete.hurlx" << 'EOF'
DELETE https://httpbin.org/delete
HTTP 200
EOF

echo "5. PATCH request"
cat > "$OUTPUT_DIR/05_patch.hurlx" << 'EOF'
PATCH https://httpbin.org/patch
Content-Type: application/json
{"field": "value"}
HTTP 200
EOF

echo "6. HEAD request"
cat > "$OUTPUT_DIR/06_head.hurlx" << 'EOF'
HEAD https://httpbin.org/get
HTTP 200
EOF

# 7. Headers
echo "7. Custom headers"
cat > "$OUTPUT_DIR/07_headers.hurlx" << 'EOF'
GET https://httpbin.org/headers
X-Test-Header: hello-world
Accept: application/json
HTTP 200
[Asserts]
jsonpath "$.headers.X-Test-Header" == "hello-world"
EOF

# 8. Query Parameters
echo "8. Query params section"
cat > "$OUTPUT_DIR/08_query_params.hurlx" << 'EOF'
GET https://httpbin.org/get
[Query]
page: 2
size: 10
HTTP 200
[Asserts]
jsonpath "$.args.page" == "2"
jsonpath "$.args.size" == "10"
EOF

# 9. Form Data
echo "9. Form POST"
cat > "$OUTPUT_DIR/09_form.hurlx" << 'EOF'
POST https://httpbin.org/post
[Form]
username: admin
password: secret123
HTTP 200
[Asserts]
jsonpath "$.form.username" == "admin"
jsonpath "$.form.password" == "secret123"
EOF

# 10. JSON Assertions
echo "10. JSON asserts"
cat > "$OUTPUT_DIR/10_json_asserts.hurlx" << 'EOF'
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

# 11. Status Code
echo "11. 404 status"
cat > "$OUTPUT_DIR/11_status_404.hurlx" << 'EOF'
GET https://httpbin.org/status/404
HTTP 404
EOF

# 12. Wildcard Status
echo "12. Wildcard status"
cat > "$OUTPUT_DIR/12_status_wildcard.hurlx" << 'EOF'
GET https://httpbin.org/status/404
HTTP *
[Asserts]
status == 404
EOF

# 13. Captures and Chaining
echo "13. Capture and chain"
cat > "$OUTPUT_DIR/13_capture_chain.hurlx" << 'EOF'
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

# 14. Filters - split, nth
echo "14. Filters (split, nth)"
cat > "$OUTPUT_DIR/14_filters.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
ua: jsonpath "$.headers.User-Agent"
ua_parts: jsonpath "$.headers.User-Agent" split "/" nth 0
[Asserts]
variable "ua_parts" exists
EOF

# 15. Count filter
echo "15. Count filter"
cat > "$OUTPUT_DIR/15_count_filter.hurlx" << 'EOF'
GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides" count > 0
jsonpath "$.slideshow.slides[0].title" isString
EOF

# 16. Basic Auth
echo "16. Basic auth"
cat > "$OUTPUT_DIR/16_basicauth.hurlx" << 'EOF'
GET https://httpbin.org/basic-auth/admin/password
[BasicAuth]
admin: password
HTTP 200
[Asserts]
jsonpath "$.authenticated" == true
EOF

# 17. Redirects
echo "17. Redirect following"
cat > "$OUTPUT_DIR/17_redirect.hurlx" << 'EOF'
GET https://httpbin.org/redirect/2
[Options]
location: true
HTTP 200
[Asserts]
url == "https://httpbin.org/get"
redirects count == 2
EOF

# 18. Bytes and Hash
echo "18. Bytes/hash asserts"
cat > "$OUTPUT_DIR/18_bytes_hash.hurlx" << 'EOF'
GET https://httpbin.org/bytes/10
HTTP 200
[Asserts]
bytes count == 10
sha256 exists
md5 exists
EOF

# 19. Duration
echo "19. Duration assert"
cat > "$OUTPUT_DIR/19_duration.hurlx" << 'EOF'
GET https://httpbin.org/delay/0.5
HTTP 200
[Asserts]
duration < 10000
EOF

# 20. Variable Assert
echo "20. Variable assert"
cat > "$OUTPUT_DIR/20_variable.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Captures]
origin: jsonpath "$.origin"
[Asserts]
variable "origin" exists
variable "origin" isString
EOF

# 21. UUID
echo "21. UUID isUuid assert"
cat > "$OUTPUT_DIR/21_uuid.hurlx" << 'EOF'
GET https://httpbin.org/uuid
HTTP 200
[Captures]
uuid: jsonpath "$.uuid"
[Asserts]
variable "uuid" isUuid
EOF

# 22. Multiple Entries
echo "22. Multiple entries"
cat > "$OUTPUT_DIR/22_multiple.hurlx" << 'EOF'
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

# 23. Not predicate
echo "23. Not predicate"
cat > "$OUTPUT_DIR/23_not.hurlx" << 'EOF'
GET https://httpbin.org/get
HTTP 200
[Asserts]
jsonpath "$.headers.X-Missing" not exists
status not == 404
EOF

# 24. Cookies
echo "24. Cookies"
cat > "$OUTPUT_DIR/24_cookies.hurlx" << 'EOF'
GET https://httpbin.org/cookies/set?test_cookie=hello
[Options]
location: true
HTTP 200

GET https://httpbin.org/cookies
HTTP 200
[Asserts]
jsonpath "$.cookies.test_cookie" == "hello"
EOF

# 25. XPath
echo "25. XPath on HTML"
cat > "$OUTPUT_DIR/25_xpath.hurlx" << 'EOF'
GET https://httpbin.org/html
HTTP 200
[Asserts]
xpath "string(//h1)" exists
xpath "//p" count > 0
EOF

echo ""
echo "=== 生成完成 ==="
echo "示例文件位置: $OUTPUT_DIR/"
echo ""
echo "运行方式:"
echo "  ./hurlx --test $OUTPUT_DIR/01_get.hurlx"
echo "  ./hurlx --test $OUTPUT_DIR/*.hurlx"
echo ""