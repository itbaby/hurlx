GET https://httpbin.org/json
HTTP 200
[Asserts]
body replace "Yours" "My" contains "My Truly"
