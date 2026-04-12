GET https://httpbin.org/anything
X-Custom: {{custom}}
[Options]
variable: custom=test-value
HTTP 200
[Asserts]
jsonpath "$.headers.X-Custom" == "test-value"