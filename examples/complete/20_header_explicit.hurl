GET https://httpbin.org/headers
HTTP 200
[Asserts]
header "Content-Type" exists
header "Content-Type" contains "json"
