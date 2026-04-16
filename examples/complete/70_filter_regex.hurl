GET https://httpbin.org/headers
HTTP 200
[Captures]
ct: header "Content-Type" regex "([\\w]+)/([\\w]+)"
[Asserts]
variable "ct" == "application"
