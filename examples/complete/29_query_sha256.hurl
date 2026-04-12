GET https://httpbin.org/bytes/10
HTTP 200
[Asserts]
sha256 exists
sha256 matches "^[a-f0-9]{64}$"
