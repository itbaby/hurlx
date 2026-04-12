GET https://httpbin.org/anything?key=hello%20world
HTTP 200
[Asserts]
url urlQueryParam "key" == "hello world"