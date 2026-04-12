GET https://httpbin.org/json
HTTP 200
[Captures]
slides: body jsonpath "$.slideshow.slides"
[Asserts]
variable "slides" isList
