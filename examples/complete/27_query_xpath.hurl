GET https://httpbin.org/html
HTTP 200
[Asserts]
xpath "string(//title)" exists
