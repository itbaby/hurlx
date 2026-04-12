GET https://httpbin.org/get
HTTP 200
[Captures]
origin: jsonpath "$.origin"
[Asserts]
variable "origin" exists
variable "origin" isString
