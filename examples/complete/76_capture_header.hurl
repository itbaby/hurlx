GET https://httpbin.org/headers
HTTP 200
[Captures]
content_type: header "Content-Type"
[Asserts]
variable "content_type" contains "json"
