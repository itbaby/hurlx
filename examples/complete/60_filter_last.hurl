GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.slides" last isObject