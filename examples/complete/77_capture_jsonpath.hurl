GET https://httpbin.org/json
HTTP 200
[Captures]
author: jsonpath "$.slideshow.author"
[Asserts]
variable "author" == "Yours Truly"
