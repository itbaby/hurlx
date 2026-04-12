GET https://httpbin.org/get
HTTP 200
[Asserts]
jsonpath "$.nonexistent" not exists
