GET https://httpbin.org/json
HTTP 200
[Captures]
count: jsonpath "$.slideshow.slides" count
[Asserts]
variable "count" toInt > 0
