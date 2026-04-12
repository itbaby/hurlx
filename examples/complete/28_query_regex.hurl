GET https://httpbin.org/uuid
HTTP 200
[Asserts]
regex "([0-9a-f-]{36})" exists
