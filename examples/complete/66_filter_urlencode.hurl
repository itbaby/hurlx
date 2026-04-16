GET https://httpbin.org/anything?key=hello world
HTTP 200
[Asserts]
url urlQueryParam "key" == "hello world"
