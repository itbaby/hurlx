GET https://httpbin.org/redirect-to?url=https://example.com
[Options]
location: true
HTTP 200
[Asserts]
redirects nth 0 location == "https://example.com"
