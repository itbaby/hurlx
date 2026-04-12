# hurlx

<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0-blue?style=flat-square" alt="Version">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License">
</p>

**hurlx** is an enhanced version of [Hurl](https://hurl.dev), designed for modern API engineering workflows.

Fully compatible with all Hurl features, hurlx **uniquely supports import/export syntax**, making HTTP testing more modular and maintainable.

---

## ✨ Key Features

### 🔄 Modular Import/Export

```hurlx
# Import common configuration
import "common/auth.hurlx"

# Import shared API endpoints
import "endpoints/users.hurlx"

# Execute tests
GET https://api.example.com/users
HTTP 200
[Asserts]
jsonpath "$.length()" > 0

# Export variables for use in other files
export token
```

**Benefits**:
- ✅ Reuse authentication, configuration, and endpoint definitions
- ✅ Better team collaboration
- ✅ Modular test cases that are easy to maintain

---

## 🚀 vs Hurl

| Feature | Hurl | hurlx |
|---------|------|-------|
| Basic HTTP Testing | ✅ | ✅ |
| JSON/XML Assertions | ✅ | ✅ |
| Variables & Templates | ✅ | ✅ |
| Filters & Predicates | ✅ | ✅ |
| **Modular Import/Export** | ❌ | ✅ |
| **Structured Test Cases** | ❌ | ✅ |
| **Engineering-Ready** | ❌ | ✅ |

> hurlx = Hurl superset + modular capabilities

---

## 📦 Features

### HTTP Methods
- GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS

### Request Configuration
- Query parameters, Form data, Multipart uploads
- Basic Auth, Bearer Token, custom Headers
- JSON/XML/Text/Base64 request body
- File uploads

### Response Assertions
- Status code assertions (wildcard `*` supported)
- Header assertions
- Body assertions (exact match, JSONPath, XPath, Regex)
- Type checks (`isString`, `isInteger`, `isList`, `isObject`, `isUuid`, `isIsoDate`, etc.)

### Advanced Features
- Variable chaining
- Conditional assertions
- Retry mechanism
- Redirect tracking
- Timeout control
- Proxy support
- HTTPS / Certificate management

---

## 🛠️ Installation

### Binary Installation

```bash
# macOS (Apple Silicon)
curl -L https://github.com/wei-lli/hurlx/releases/latest/download/hurlx-1.0.0-darwin-arm64 -o hurlx
chmod +x hurlx

# macOS (Intel)
curl -L https://github.com/wei-lli/hurlx/releases/latest/download/hurlx-1.0.0-darwin-amd64 -o hurlx
chmod +x hurlx

# Linux
curl -L https://github.com/wei-lli/hurlx/releases/latest/download/hurlx-1.0.0-linux-amd64 -o hurlx
chmod +x hurlx

# Windows
curl -L https://github.com/wei-lli/hurlx/releases/latest/download/hurlx-1.0.0-windows-amd64.exe -o hurlx.exe
```

### Go Install

```bash
go install github.com/wei-lli/hurlx/cli
```

### Build from Source

```bash
git clone https://github.com/wei-lli/hurlx.git
cd hurlx
go build -o hurlx ./cli
```

---

## 📖 Quick Start

### Basic Usage

```hurlx
# hello.hurlx
GET https://example.com
HTTP 200
```

```bash
hurlx hello.hurlx
```

### Test Mode

```bash
hurlx --test hello.hurlx
```

### Using Variables

```bash
hurlx --variable host=api.example.com api.hurlx

# or load from file
hurlx --variables-file env.json api.hurlx
```

### Import/Export Example

```hurlx
# login.hurlx - Login and export token
POST https://api.example.com/login
[JSON]
{
  "username": "admin",
  "password": "admin"
}
HTTP 200
[Captures]
token: jsonpath "$.token"

# Export token for other files
export token
```

```hurlx
# main.hurlx - Import and use
import "login.hurlx"

GET https://api.example.com/users
Authorization: Bearer {{token}}
HTTP 200
```

---


## 🔧 CLI Options

```
hurlx [options] [FILE...]

Options:
  -4                      Use IPv4 only
  -6                      Use IPv6 only
  -L, -location           Follow redirects
  -V value                Define variable
  --compressed            Request compressed response
  --connect-timeout       Connection timeout
  --continue-on-error     Continue on assert errors
  --insecure, -k          Allow insecure SSL connections
  --json                  JSON output
  --test                  Test mode (assertions only)
  --timeout, -m           Maximum time per request
  --retry                 Retry count
  --variable, -V          Define variable
  --variables-file       Load variables from file
  --verbose, -v          Verbose output
  --very-verbose         More verbose output
  -i, --include          Include HTTP headers in output
  -o, --output           Output file
```

---

## 📊 Test Coverage

| Category | Coverage |
|----------|----------|
| HTTP Methods | 100% |
| Request Sections | 100% |
| Request Body Types | 100% |
| Response Assertions | 100% |
| Queries (JSONPath, XPath, Regex, etc.) | 100% |
| Predicates | 100% |
| Filters | 95%+ |
| Templates | 100% |
| Captures | 100% |
| Import/Export | 100% |

---

## 🤝 Contributing

Issues and Pull Requests are welcome!

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- [Hurl](https://hurl.dev) - Powerful HTTP testing tool
- All contributors and test users