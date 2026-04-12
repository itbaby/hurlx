GET https://httpbin.org/get
HTTP 200
[Captures]
date: jsonpath "$.origin"
[Asserts]
variable "date" exists
