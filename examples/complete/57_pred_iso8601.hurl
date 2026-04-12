GET https://httpbin.org/get
HTTP 200
[Captures]
date: header "Date"
[Asserts]
variable "date" isIsoDate
