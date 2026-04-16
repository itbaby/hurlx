GET https://httpbin.org/headers
HTTP 200
[Asserts]
jsonpath "$.headers.Host" exists
