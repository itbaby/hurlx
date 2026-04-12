GET https://httpbin.org/get
[Query]
key1: value1
key2: value2
HTTP 200
[Asserts]
jsonpath "$.args.key1" == "value1"
jsonpath "$.args.key2" == "value2"
