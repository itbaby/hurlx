# Hurlx 示例测试用例

所有 25 个示例已验证通过，运行命令：

```bash
./hurlx --test examples/tests/*.hurlx
```

## 测试用例列表

| # | 文件 | 说明 |
|---|------|------|
| 01 | `01_get.hurlx` | GET 请求 |
| 02 | `02_post_json.hurlx` | POST JSON body |
| 03 | `03_put.hurlx` | PUT 请求 |
| 04 | `04_delete.hurlx` | DELETE 请求 |
| 05 | `05_patch.hurlx` | PATCH 请求 |
| 06 | `06_head.hurlx` | HEAD 请求 |
| 07 | `07_headers.hurlx` | 自定义请求头 |
| 08 | `08_query_params.hurlx` | Query 参数 section |
| 09 | `09_form.hurlx` | Form POST |
| 10 | `10_json_asserts.hurlx` | JSON 断言（多个断言类型） |
| 11 | `11_status_404.hurlx` | 404 状态码 |
| 12 | `12_status_wildcard.hurlx` | 状态码通配符 `HTTP *` |
| 13 | `13_capture_chain.hurlx` | 捕获 + 变量链化 |
| 14 | `14_filters.hurlx` | 过滤器 (split, nth) |
| 15 | `15_count_filter.hurlx` | count 过滤器 |
| 16 | `16_basicauth.hurlx` | Basic Auth |
| 17 | `17_redirect.hurlx` | 跟随重定向 + redirects count |
| 18 | `18_bytes_hash.hurlx` | bytes 计数 + sha256/md5 |
| 19 | `19_duration.hurlx` | duration 断言 |
| 20 | `20_variable.hurlx` | variable 断言 |
| 21 | `21_uuid.hurlx` | UUID + isUuid 断言 |
| 22 | `22_multiple.hurlx` | 多条目链式请求 |
| 23 | `23_not.hurlx` | not 谓词 (not exists, not ==) |
| 24 | `24_cookies.hurlx` | Cookies + 重定向 |
| 25 | `25_xpath.hurlx` | XPath 断言 (HTML) |

## 使用示例

### 1. 简单 GET 请求
```bash
./hurlx --test examples/tests/01_get.hurlx
```

### 2. 批量测试
```bash
./hurlx --test examples/tests/*.hurlx
```

### 3. 带变量的测试
```bash
./hurlx --test examples/tests/13_capture_chain.hurlx -V token=abc123
```

### 4. 查看详细输出
```bash
./hurlx --test examples/tests/10_json_asserts.hurlx -v
```

### 5. JSON 输出
```bash
./hurlx --test examples/tests/01_get.hurlx --json
```