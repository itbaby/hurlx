GET https://httpbin.org/redirect/2
[Options]
location: true
HTTP 200
[Asserts]
redirects count == 2
