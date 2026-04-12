GET https://httpbin.org/headers
HTTP 200
[Captures]
ct: header "Content-Type" regex "([a-z]+)/([a-z]+)"
[Asserts]
variable "ct" == "application"