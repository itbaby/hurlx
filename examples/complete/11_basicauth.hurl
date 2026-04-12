GET https://httpbin.org/basic-auth/user/passwd
[BasicAuth]
user: passwd
HTTP 200
[Asserts]
jsonpath "$.authenticated" == true
