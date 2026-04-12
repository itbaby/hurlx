GET https://httpbin.org/headers
[Options]
user-agent: TestAgent
HTTP 200
[Asserts]
jsonpath "$.headers.User-Agent" == "TestAgent"