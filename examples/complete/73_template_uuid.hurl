GET https://httpbin.org/uuid
HTTP 200
[Captures]
uuid: jsonpath "$.uuid"
[Asserts]
variable "uuid" isUuid
