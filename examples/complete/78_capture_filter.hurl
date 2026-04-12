GET https://httpbin.org/json
HTTP 200
[Captures]
author: jsonpath "$.slideshow.author" toString upper
[Asserts]
variable "author" == "YOURS TRULY"
