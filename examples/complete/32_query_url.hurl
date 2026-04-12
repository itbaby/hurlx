GET https://httpbin.org/redirect/1
[Options]
location: true
HTTP 200
[Asserts]
url == "https://httpbin.org/get"
