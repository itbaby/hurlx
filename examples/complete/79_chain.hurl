GET https://httpbin.org/uuid
HTTP 200
[Captures]
uuid: jsonpath "$.uuid"

GET https://httpbin.org/anything
X-Request-Id: {{uuid}}
HTTP 200
