GET https://httpbin.org/headers
HTTP 200
[Captures]
ct: header "Content-Type" split "/" nth 0
[Asserts]
variable "ct" == "application"
