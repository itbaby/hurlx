POST https://httpbin.org/post
[Form]
username: admin
password: secret
HTTP 200
[Asserts]
jsonpath "$.form.username" == "admin"
