GET https://httpbin.org/json
HTTP 200
[Asserts]
jsonpath "$.slideshow.author" matches "Yours .*"
