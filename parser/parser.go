package parser

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/wei-lli/hurlx/ast"
)

var lastTagRegex = regexp.MustCompile(`</\w+>`)

var responseStatusRegex = regexp.MustCompile(`^HTTP(?:/([\d.]+))?\s+(\d+|\*)`)

type Parser struct {
	scanner  *bufio.Scanner
	line     int
	col      int
	lineStr  string
	pos      int
	file     string
	bufLine  string
	bufReady bool
}

func NewParser(input string, filePath string) *Parser {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	return &Parser{
		scanner: scanner,
		file:    filePath,
		line:    0,
	}
}

func (p *Parser) Parse() (*ast.File, error) {
	file := &ast.File{}

	for {
		if p.pos >= len(p.lineStr) || p.lineStr == "" {
			if !p.nextLine() {
				break
			}
		}

		p.skipEmptyAndComments(file)

		if p.pos >= len(p.lineStr) {
			continue
		}

		line := p.currentLine()

		if strings.HasPrefix(line, "import ") {
			imp, err := p.parseImport()
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, *imp)
			p.pos = len(p.lineStr)
			continue
		}

		if strings.HasPrefix(line, "export ") {
			exp, err := p.parseExport()
			if err != nil {
				return nil, err
			}
			file.Exports = append(file.Exports, *exp)
			p.pos = len(p.lineStr)
			continue
		}

		if isMethod(line) {
			entry, err := p.parseEntry()
			if err != nil {
				return nil, err
			}
			file.Entries = append(file.Entries, *entry)
			continue
		}

		p.pos = len(p.lineStr)
	}

	return file, nil
}

func (p *Parser) nextLine() bool {
	if p.bufReady {
		p.bufReady = false
		p.pos = 0
		return true
	}
	if !p.scanner.Scan() {
		return false
	}
	p.line++
	p.lineStr = p.scanner.Text()
	p.pos = 0
	return true
}

func (p *Parser) unreadLine() {
	p.bufLine = p.lineStr
	p.bufReady = true
}

func (p *Parser) currentLine() string {
	if p.pos >= len(p.lineStr) {
		return ""
	}
	return strings.TrimSpace(p.lineStr[p.pos:])
}

func (p *Parser) rawLine() string {
	return p.lineStr
}

func (p *Parser) skipEmptyAndComments(file *ast.File) {
	for {
		line := p.currentLine()
		if line == "" {
			if strings.HasPrefix(p.rawLine(), "#") {
				file.Comments = append(file.Comments, ast.Comment{
					Text:     strings.TrimPrefix(p.rawLine(), "#"),
					Position: ast.Position{Line: p.line},
				})
			}
			p.pos = len(p.lineStr)
			return
		}
		if strings.HasPrefix(line, "#") {
			file.Comments = append(file.Comments, ast.Comment{
				Text:     line[1:],
				Position: ast.Position{Line: p.line},
			})
			p.pos = len(p.lineStr)
			return
		}
		return
	}
}

func (p *Parser) parseImport() (*ast.Import, error) {
	line := p.currentLine()
	line = strings.TrimPrefix(line, "import ")
	line = strings.TrimSpace(line)

	var alias, path string
	if idx := strings.Index(line, " as "); idx >= 0 {
		path = strings.TrimSpace(line[:idx])
		alias = strings.TrimSpace(line[idx+4:])
	} else {
		path = strings.TrimSpace(line)
	}

	path = strings.Trim(path, "\"'`")
	imp := &ast.Import{
		Path:     path,
		Alias:    alias,
		Position: ast.Position{Line: p.line},
	}
	return imp, nil
}

func (p *Parser) parseExport() (*ast.Export, error) {
	line := p.currentLine()
	line = strings.TrimPrefix(line, "export ")
	line = strings.TrimSpace(line)

	parts := strings.SplitN(line, "=", 2)
	name := strings.TrimSpace(parts[0])
	var value string
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}
	value = strings.Trim(value, "\"'`")

	return &ast.Export{
		Name:     name,
		Value:    value,
		Position: ast.Position{Line: p.line},
	}, nil
}

func (p *Parser) parseEntry() (*ast.Entry, error) {
	entry := &ast.Entry{}

	req, err := p.parseRequest()
	if err != nil {
		return nil, err
	}
	entry.Request = req

	line := p.currentLine()
	if line != "" && isResponseStatus(line) {
		resp, err := p.parseResponse()
		if err != nil {
			return nil, err
		}
		entry.Response = resp
		return entry, nil
	}

	if line == "" {
		savedLine := p.lineStr
		savedPos := p.pos
		for p.nextLine() {
			next := p.currentLine()
			if next == "" {
				continue
			}
			if isResponseStatus(next) {
				resp, err := p.parseResponse()
				if err != nil {
					return nil, err
				}
				entry.Response = resp
				return entry, nil
			}
			p.lineStr = savedLine
			p.pos = savedPos
			return entry, nil
		}
	}

	return entry, nil
}

func (p *Parser) parseRequest() (*ast.Request, error) {
	req := &ast.Request{}

	line := p.currentLine()
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("line %d: invalid request line: %s", p.line, line)
	}
	req.Method = parts[0]
	req.URL = parts[1]
	p.pos = len(p.lineStr)

	for {
		if p.pos >= len(p.lineStr) {
			if !p.nextLine() {
				break
			}
		}
		line := p.currentLine()
		if line == "" || isMethod(line) || isResponseStatus(line) || strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "export ") {
			return req, nil
		}

		switch {
		case line == "[Options]":
			req.Options = p.parseOptionsSection()
			p.pos = len(p.lineStr)
		case line == "[QueryStringParams]" || line == "[Query]":
			req.Query = p.parseKeyValueSection()
			p.pos = len(p.lineStr)
		case line == "[FormParams]" || line == "[Form]":
			req.Form = p.parseKeyValueSection()
			p.pos = len(p.lineStr)
		case line == "[Multipart]" || line == "[MultipartFormData]":
			req.Multipart = p.parseMultipartSection()
			p.pos = len(p.lineStr)
		case line == "[Cookies]":
			req.Cookies = p.parseKeyValueSection()
			p.pos = len(p.lineStr)
		case line == "[BasicAuth]":
			req.BasicAuth = p.parseBasicAuth()
			p.pos = len(p.lineStr)
		case isSection(line):
			return req, nil
		case isBodyStart(line) || looksLikeBody(line):
			body, err := p.parseBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
			p.pos = len(p.lineStr)
			return req, nil
		case strings.Contains(line, ":") && !strings.HasPrefix(line, "["):
			hdr := p.parseHeader(line)
			req.Headers = append(req.Headers, hdr)
			p.pos = len(p.lineStr)
		default:
			p.pos = len(p.lineStr)
		}
	}

	return req, nil
}

func (p *Parser) parseHeader(line string) ast.Header {
	idx := strings.Index(line, ":")
	name := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return ast.Header{Name: name, Value: value}
}

func (p *Parser) parseKeyValueSection() []ast.KeyValue {
	var kvs []ast.KeyValue
	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isSection(line) || isMethod(line) || isResponseStatus(line) || strings.HasPrefix(line, "[") {
			p.unreadLine()
			return kvs
		}
		idx := strings.Index(line, ":")
		if idx >= 0 {
			kvs = append(kvs, ast.KeyValue{
				Key:   strings.TrimSpace(line[:idx]),
				Value: strings.TrimSpace(line[idx+1:]),
			})
		}
	}
	return kvs
}

func (p *Parser) parseMultipartSection() []ast.MultipartField {
	var fields []ast.MultipartField
	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isSection(line) || isMethod(line) || isResponseStatus(line) {
			p.unreadLine()
			return fields
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		field := ast.MultipartField{Name: name}
		if strings.HasPrefix(value, "file,") {
			field.IsFile = true
			filePart := strings.TrimPrefix(value, "file,")
			filePart = strings.TrimSuffix(filePart, ";")
			parts := strings.SplitN(filePart, ";", 2)
			field.Value = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				field.FileType = strings.TrimSpace(parts[1])
			}
		} else {
			field.Value = value
		}
		fields = append(fields, field)
	}
	return fields
}

func (p *Parser) parseBasicAuth() *ast.BasicAuth {
	if !p.nextLine() {
		return nil
	}
	line := p.currentLine()
	if line == "" || isSection(line) {
		return nil
	}
	idx := strings.Index(line, ":")
	if idx < 0 {
		return &ast.BasicAuth{Username: line}
	}
	return &ast.BasicAuth{
		Username: strings.TrimSpace(line[:idx]),
		Password: strings.TrimSpace(line[idx+1:]),
	}
}

func (p *Parser) parseOptionsSection() *ast.OptionsSection {
	opts := &ast.OptionsSection{
		Variables: make(map[string]string),
	}
	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isSection(line) || isMethod(line) || isResponseStatus(line) {
			p.unreadLine()
			return opts
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		switch key {
		case "location", "follow-redirect", "follow_redirect":
			opts.Location = parseBoolPtr(val)
		case "max-redirs", "max_redirs":
			if n, err := strconv.Atoi(val); err == nil {
				opts.MaxRedirs = &n
			}
		case "insecure":
			opts.Insecure = parseBoolPtr(val)
		case "verbose":
			opts.Verbose = parseBoolPtr(val)
		case "compressed":
			opts.Compressed = parseBoolPtr(val)
		case "retry":
			if n, err := strconv.Atoi(val); err == nil {
				opts.Retry = &n
			}
		case "retry-interval", "retry_interval":
			opts.RetryInterval = val
		case "max-time", "max_time", "timeout":
			opts.Timeout = val
		case "connect-timeout", "connect_timeout":
			opts.ConnectTimeout = val
		case "delay":
			opts.Delay = val
		case "skip":
			opts.Skip = parseBoolPtr(val)
		case "output":
			opts.Output = val
		case "variable":
			if vp := strings.SplitN(val, "=", 2); len(vp) == 2 {
				opts.Variables[vp[0]] = vp[1]
			}
		case "proxy":
			opts.Proxy = val
		case "user":
			opts.User = val
		case "user-agent", "user_agent", "useragent":
			opts.UserAgent = val
		case "http3":
			opts.HTTP3 = parseBoolPtr(val)
		case "cacert":
			opts.CACert = val
		case "cert":
			opts.Cert = val
		case "key":
			opts.Key = val
		case "aws-sigv4":
			opts.AWSSigV4 = val
		case "ipv4":
			opts.IPv4 = parseBoolPtr(val)
		case "ipv6":
			opts.IPv6 = parseBoolPtr(val)
		case "limit-rate", "limit_rate":
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				opts.LimitRate = &n
			}
		case "path-as-is", "path_as_is":
			opts.PathAsIs = parseBoolPtr(val)
		case "unix-socket", "unix_socket":
			opts.UnixSocket = val
		}
	}
	return opts
}

func (p *Parser) parseBody() (*ast.Body, error) {
	line := p.currentLine()

	if strings.HasPrefix(line, "```") {
		return p.parseMultilineBody()
	}

	if strings.HasPrefix(line, "`") && strings.HasSuffix(line, "`") && strings.Count(line, "`") == 2 {
		return &ast.Body{
			Type:    ast.BodyOneline,
			Content: line[1 : len(line)-1],
		}, nil
	}

	if strings.HasPrefix(line, "base64,") {
		content := strings.TrimPrefix(line, "base64,")
		content = strings.TrimSuffix(content, ";")
		return &ast.Body{Type: ast.BodyBase64, Content: content}, nil
	}

	if strings.HasPrefix(line, "hex,") {
		content := strings.TrimPrefix(line, "hex,")
		content = strings.TrimSuffix(content, ";")
		return &ast.Body{Type: ast.BodyHex, Content: content}, nil
	}

	if strings.HasPrefix(line, "file,") {
		content := strings.TrimPrefix(line, "file,")
		content = strings.TrimSuffix(content, ";")
		return &ast.Body{Type: ast.BodyFile, Content: content}, nil
	}

	if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
		return p.parseJSONBody()
	}

	if strings.HasPrefix(line, "<?xml") {
		return p.parseXMLBody()
	}

	return nil, fmt.Errorf("line %d: unsupported body type: %s", p.line, line)
}

func (p *Parser) parseJSONBody() (*ast.Body, error) {
	var sb strings.Builder
	braceCount := 0
	bracketCount := 0

	for {
		line := p.rawLine()
		trimmed := strings.TrimSpace(line)

		for _, ch := range trimmed {
			switch ch {
			case '{':
				braceCount++
			case '}':
				braceCount--
			case '[':
				bracketCount++
			case ']':
				bracketCount--
			}
		}

		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)

		if braceCount <= 0 && bracketCount <= 0 {
			break
		}

		if !p.nextLine() {
			break
		}
	}

	return &ast.Body{
		Type:    ast.BodyJSON,
		Content: sb.String(),
	}, nil
}

func (p *Parser) parseXMLBody() (*ast.Body, error) {
	var sb strings.Builder
	tagCount := 0
	inTag := false

	for {
		line := p.rawLine()
		trimmed := strings.TrimSpace(line)

		for i, ch := range trimmed {
			if ch == '<' && i+1 < len(trimmed) && trimmed[i+1] != '/' && !strings.HasPrefix(trimmed[i:], "<!") && !strings.HasPrefix(trimmed[i:], "<?") {
				tagCount++
				inTag = true
			}
			if ch == '>' && inTag {
				inTag = false
			}
			if ch == '<' && i+1 < len(trimmed) && trimmed[i+1] == '/' {
				tagCount--
			}
		}

		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)

		if tagCount <= 0 && sb.Len() > 0 && strings.Contains(trimmed, ">") {
			if lastTagRegex.MatchString(trimmed) {
				break
			}
		}

		if !p.nextLine() {
			break
		}
	}

	return &ast.Body{
		Type:    ast.BodyXML,
		Content: sb.String(),
	}, nil
}

func (p *Parser) parseMultilineBody() (*ast.Body, error) {
	line := p.currentLine()
	lang := ""
	if len(line) > 3 {
		lang = strings.TrimSpace(line[3:])
	}

	var sb strings.Builder
	for p.nextLine() {
		line := p.currentLine()
		if line == "```" {
			break
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(p.rawLine())
	}

	bodyType := ast.BodyMultiline
	switch lang {
	case "json":
		bodyType = ast.BodyJSON
	case "xml":
		bodyType = ast.BodyXML
	}

	return &ast.Body{
		Type:    bodyType,
		Content: sb.String(),
		Lang:    lang,
	}, nil
}

func (p *Parser) parseResponse() (*ast.Response, error) {
	resp := &ast.Response{}

	line := p.currentLine()
	version, status, err := parseResponseStatus(line)
	if err != nil {
		return nil, err
	}
	resp.Version = version
	resp.Status = status

	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isMethod(line) || strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "export ") {
			return resp, nil
		}

		switch {
		case line == "[Captures]":
			captures, err := p.parseCaptures()
			if err != nil {
				return nil, err
			}
			resp.Captures = captures
			p.pos = len(p.lineStr)
		case line == "[Asserts]":
			asserts, err := p.parseAsserts()
			if err != nil {
				return nil, err
			}
			resp.Asserts = asserts
			p.pos = len(p.lineStr)
		case isSection(line):
			return resp, nil
		case isBodyStart(line):
			body, err := p.parseBody()
			if err != nil {
				return nil, err
			}
			resp.Body = body
			return resp, nil
		case strings.Contains(line, ":") && !strings.HasPrefix(line, "["):
			hdr := p.parseHeader(line)
			resp.Headers = append(resp.Headers, hdr)
		default:
			if looksLikeBody(line) {
				body, err := p.parseBody()
				if err != nil {
					return nil, err
				}
				resp.Body = body
				return resp, nil
			}
		}
	}

	return resp, nil
}

func (p *Parser) parseCaptures() ([]ast.Capture, error) {
	var captures []ast.Capture
	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isSection(line) || isMethod(line) || isResponseStatus(line) {
			p.unreadLine()
			return captures, nil
		}

		cap, err := p.parseCapture(line)
		if err != nil {
			return nil, err
		}
		captures = append(captures, *cap)
	}
	return captures, nil
}

func (p *Parser) parseCapture(line string) (*ast.Capture, error) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return nil, fmt.Errorf("line %d: invalid capture: %s", p.line, line)
	}

	varName := strings.TrimSpace(line[:idx])
	rest := strings.TrimSpace(line[idx+1:])

	query, filters, rest, err := p.parseQueryAndFilters(rest)
	if err != nil {
		return nil, err
	}

	redact := false
	rest = strings.TrimSpace(rest)
	if rest == "redact" || strings.HasSuffix(rest, " redact") {
		redact = true
	}

	return &ast.Capture{
		Variable: varName,
		Query:    query,
		Filters:  filters,
		Redact:   redact,
	}, nil
}

func (p *Parser) parseAsserts() ([]ast.Assert, error) {
	var asserts []ast.Assert
	for p.nextLine() {
		line := p.currentLine()
		if line == "" || isSection(line) || isMethod(line) || isResponseStatus(line) {
			p.unreadLine()
			return asserts, nil
		}

		a, err := p.parseAssert(line)
		if err != nil {
			return nil, err
		}
		asserts = append(asserts, *a)
	}
	return asserts, nil
}

func (p *Parser) parseAssert(line string) (*ast.Assert, error) {
	query, filters, rest, err := p.parseQueryAndFilters(line)
	if err != nil {
		return nil, err
	}

	not := false
	if strings.HasPrefix(rest, "not ") {
		not = true
		rest = strings.TrimPrefix(rest, "not ")
	}

	predType, value, err := parsePredicate(rest)
	if err != nil {
		return nil, err
	}

	return &ast.Assert{
		Query:     query,
		Filters:   filters,
		Not:       not,
		Predicate: predType,
		Value:     value,
	}, nil
}

func (p *Parser) parseQueryAndFilters(input string) (ast.Query, []ast.Filter, string, error) {
	input = strings.TrimSpace(input)
	var query ast.Query
	var filters []ast.Filter

	queryTypes := []struct {
		prefix string
		qtype  ast.QueryType
	}{
		{"jsonpath", ast.QueryJSONPath},
		{"xpath", ast.QueryXPath},
		{"header", ast.QueryHeader},
		{"cookie", ast.QueryCookie},
		{"regex", ast.QueryRegex},
		{"variable", ast.QueryVariable},
		{"certificate", ast.QueryCertificate},
	}

	simpleQueries := map[string]ast.QueryType{
		"status":    ast.QueryStatus,
		"version":   ast.QueryVersion,
		"body":      ast.QueryBody,
		"bytes":     ast.QueryBytes,
		"sha256":    ast.QuerySHA256,
		"md5":       ast.QueryMD5,
		"url":       ast.QueryURL,
		"redirects": ast.QueryRedirects,
		"ip":        ast.QueryIP,
		"duration":  ast.QueryDuration,
	}

	matched := false
	for _, qt := range queryTypes {
		if strings.HasPrefix(input, qt.prefix+" ") || strings.HasPrefix(input, qt.prefix+"\"") {
			query.Type = qt.qtype
			rest := strings.TrimSpace(input[len(qt.prefix):])
			if strings.HasPrefix(rest, "\"") {
				val, remaining := extractQuotedString(rest)
				query.Value = val
				input = remaining
			} else if strings.HasPrefix(rest, "/") {
				val, remaining := extractRegexPattern(rest)
				query.Value = val
				input = remaining
			} else {
				parts := strings.SplitN(rest, " ", 2)
				query.Value = parts[0]
				if len(parts) > 1 {
					input = parts[1]
				} else {
					input = ""
				}
			}
			matched = true
			break
		}
	}

	if !matched {
		for prefix, qtype := range simpleQueries {
			if input == prefix || strings.HasPrefix(input, prefix+" ") {
				query.Type = qtype
				input = strings.TrimSpace(strings.TrimPrefix(input, prefix))
				matched = true
				break
			}
		}
	}

	if !matched {
		return query, nil, input, fmt.Errorf("line %d: unknown query: %s", p.line, input)
	}

	input = strings.TrimSpace(input)

	filterNames := []string{
		"base64UrlSafeDecode", "base64UrlSafeEncode",
		"base64Decode", "base64Encode",
		"daysAfterNow", "daysBeforeNow",
		"urlQueryParam",
		"replaceRegex",
		"urlDecode", "urlEncode",
		"utf8Decode", "utf8Encode",
		"toHex", "toFloat", "toInt", "toString",
		"toDate", "dateFormat",
		"htmlEscape", "htmlUnescape",
		"jsonpath", "xpath",
		"location",
		"count", "regex", "split", "first", "last",
		"decode", "replace", "nth",
	}

	noArgFilters := map[string]bool{
		"count": true, "first": true, "last": true,
		"toInt": true, "toFloat": true, "toString": true,
		"base64Decode": true, "base64Encode": true,
		"base64UrlSafeDecode": true, "base64UrlSafeEncode": true,
		"urlDecode": true, "urlEncode": true,
		"toHex": true, "htmlEscape": true, "htmlUnescape": true,
		"utf8Decode": true, "utf8Encode": true,
		"location": true, "daysAfterNow": true, "daysBeforeNow": true,
	}

	for {
		matched := false
		for _, fn := range filterNames {
			if input == fn {
				input = ""
				matched = true
				ft := filterNameToType(fn)
				filters = append(filters, ast.Filter{Type: ft})
				break
			}
			if noArgFilters[fn] {
				if strings.HasPrefix(input, fn+" ") {
					input = strings.TrimSpace(input[len(fn):])
					matched = true
					filters = append(filters, ast.Filter{Type: filterNameToType(fn)})
					break
				}
				continue
			}
			if strings.HasPrefix(input, fn+" ") || strings.HasPrefix(input, fn+"\"") {
				ft := filterNameToType(fn)
				rest := input[len(fn):]
				rest = strings.TrimSpace(rest)

				if fn == "replace" || fn == "replaceRegex" {
					if strings.HasPrefix(rest, "\"") {
						val1, remaining := extractQuotedString(rest)
						remaining = strings.TrimSpace(remaining)
						var val2 string
						if strings.HasPrefix(remaining, "\"") {
							val2, remaining = extractQuotedString(remaining)
						} else {
							parts := strings.SplitN(remaining, " ", 2)
							val2 = parts[0]
							if len(parts) > 1 {
								remaining = parts[1]
							} else {
								remaining = ""
							}
						}
						filters = append(filters, ast.Filter{Type: ft, Value: val1 + " " + val2})
						input = remaining
					} else if strings.HasPrefix(rest, "/") {
						val, remaining := extractRegexPattern(rest)
						remaining = strings.TrimSpace(remaining)
						var val2 string
						if strings.HasPrefix(remaining, "\"") {
							val2, remaining = extractQuotedString(remaining)
						} else {
							parts := strings.SplitN(remaining, " ", 2)
							val2 = parts[0]
							if len(parts) > 1 {
								remaining = parts[1]
							} else {
								remaining = ""
							}
						}
						filters = append(filters, ast.Filter{Type: ft, Value: val + " " + val2})
						input = remaining
					} else {
						parts := strings.SplitN(rest, " ", 3)
						if len(parts) >= 2 {
							filters = append(filters, ast.Filter{Type: ft, Value: parts[0] + " " + parts[1]})
							if len(parts) > 2 {
								input = parts[2]
							} else {
								input = ""
							}
						}
					}
				} else if strings.HasPrefix(rest, "\"") {
					val, remaining := extractQuotedString(rest)
					filters = append(filters, ast.Filter{Type: ft, Value: val})
					input = remaining
				} else if strings.HasPrefix(rest, "/") {
					val, remaining := extractRegexPattern(rest)
					filters = append(filters, ast.Filter{Type: ft, Value: val})
					input = remaining
				} else {
					parts := strings.SplitN(rest, " ", 2)
					filters = append(filters, ast.Filter{Type: ft, Value: parts[0]})
					if len(parts) > 1 {
						input = parts[1]
					} else {
						input = ""
					}
				}
				input = strings.TrimSpace(input)
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}

	return query, filters, input, nil
}

func parsePredicate(input string) (ast.PredicateType, ast.AssertValue, error) {
	input = strings.TrimSpace(input)

	predicatesWithValue := []struct {
		name string
		pt   ast.PredicateType
	}{
		{"startsWith", ast.PredStartsWith},
		{"endsWith", ast.PredEndsWith},
		{"contains", ast.PredContains},
		{"matches", ast.PredMatches},
		{"includes", ast.PredIncludes},
	}

	for _, pred := range predicatesWithValue {
		if strings.HasPrefix(input, pred.name+" ") || strings.HasPrefix(input, pred.name+"\"") {
			valStr := strings.TrimSpace(input[len(pred.name):])
			return pred.pt, parseValue(valStr), nil
		}
	}

	predicates := []struct {
		name string
		pt   ast.PredicateType
	}{
		{"exists", ast.PredExists},
		{"isBoolean", ast.PredIsBoolean},
		{"isEmpty", ast.PredIsEmpty},
		{"isFloat", ast.PredIsFloat},
		{"isInteger", ast.PredIsInteger},
		{"isIpv4", ast.PredIsIpv4},
		{"isIpv6", ast.PredIsIpv6},
		{"isIsoDate", ast.PredIsIsoDate},
		{"isList", ast.PredIsList},
		{"isNumber", ast.PredIsNumber},
		{"isObject", ast.PredIsObject},
		{"isString", ast.PredIsString},
		{"isUuid", ast.PredIsUuid},
		{"includes", ast.PredIncludes},
		{"isCollection", ast.PredIsCollection},
		{"isDate", ast.PredIsDate},
	}

	for _, pred := range predicates {
		if input == pred.name {
			return pred.pt, ast.AssertValue{}, nil
		}
	}

	opMap := map[string]ast.PredicateType{
		"==": ast.PredEqual,
		"!=": ast.PredNotEqual,
		">=": ast.PredGreaterEqual,
		"<=": ast.PredLessEqual,
		">":  ast.PredGreaterThan,
		"<":  ast.PredLessThan,
	}

	longFirst := []string{">=", "<=", "==", "!=", ">", "<"}
	for _, op := range longFirst {
		pt := opMap[op]
		if strings.HasPrefix(input, op) {
			valStr := strings.TrimSpace(input[len(op):])
			val := parseValue(valStr)
			return pt, val, nil
		}
	}

	return ast.PredEqual, ast.AssertValue{}, fmt.Errorf("unknown predicate: %s", input)
}

func parseValue(s string) ast.AssertValue {
	s = strings.TrimSpace(s)

	if s == "null" {
		return ast.AssertValue{Type: ast.ValueNull, IsNull: true}
	}
	if s == "true" {
		return ast.AssertValue{Type: ast.ValueBool, Bool: true}
	}
	if s == "false" {
		return ast.AssertValue{Type: ast.ValueBool, Bool: false}
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return ast.AssertValue{Type: ast.ValueString, Str: s[1 : len(s)-1]}
	}
	if strings.HasPrefix(s, "/") {
		val, _ := extractRegexPattern(s)
		return ast.AssertValue{Type: ast.ValueString, Str: val}
	}
	if strings.HasPrefix(s, "hex,") {
		hexStr := strings.TrimPrefix(s, "hex,")
		hexStr = strings.TrimSuffix(hexStr, ";")
		return ast.AssertValue{Type: ast.ValueBytes, Bytes: []byte(hexStr)}
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ast.AssertValue{Type: ast.ValueInt, Int: n}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return ast.AssertValue{Type: ast.ValueFloat, Float: f}
	}
	return ast.AssertValue{Type: ast.ValueString, Str: s}
}

func parseResponseStatus(line string) (string, int, error) {
	matches := responseStatusRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", 0, fmt.Errorf("invalid response status: %s", line)
	}

	version := ""
	if matches[1] != "" {
		version = matches[1]
	}

	status := 0
	if matches[2] != "*" {
		var err error
		status, err = strconv.Atoi(matches[2])
		if err != nil {
			return "", 0, err
		}
	}

	return version, status, nil
}

func extractQuotedString(input string) (string, string) {
	if !strings.HasPrefix(input, "\"") {
		return "", input
	}
	escaped := false
	for i := 1; i < len(input); i++ {
		if input[i] == '\\' && !escaped {
			escaped = true
			continue
		}
		if input[i] == '"' && !escaped {
			return input[1:i], input[i+1:]
		}
		escaped = false
	}
	return input[1:], ""
}

func extractRegexPattern(input string) (string, string) {
	if !strings.HasPrefix(input, "/") {
		return "", input
	}
	escaped := false
	for i := 1; i < len(input); i++ {
		if input[i] == '\\' && !escaped {
			escaped = true
			continue
		}
		if input[i] == '/' && !escaped {
			return input[1:i], input[i+1:]
		}
		escaped = false
	}
	return input[1:], ""
}

func parseBoolPtr(s string) *bool {
	s = strings.TrimSpace(s)
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &b
}

var filterNameToTypeMap = map[string]ast.FilterType{
	"count":               ast.FilterCount,
	"regex":               ast.FilterRegex,
	"replace":             ast.FilterReplace,
	"replaceRegex":        ast.FilterReplaceRegex,
	"split":               ast.FilterSplit,
	"nth":                 ast.FilterNth,
	"first":               ast.FilterFirst,
	"last":                ast.FilterLast,
	"toInt":               ast.FilterToInt,
	"toFloat":             ast.FilterToFloat,
	"toString":            ast.FilterToString,
	"toDate":              ast.FilterToDate,
	"dateFormat":          ast.FilterDateFormat,
	"daysAfterNow":        ast.FilterDaysAfterNow,
	"daysBeforeNow":       ast.FilterDaysBeforeNow,
	"base64Decode":        ast.FilterBase64Decode,
	"base64Encode":        ast.FilterBase64Encode,
	"base64UrlSafeDecode": ast.FilterBase64UrlSafeDecode,
	"base64UrlSafeEncode": ast.FilterBase64UrlSafeEncode,
	"decode":              ast.FilterDecode,
	"urlDecode":           ast.FilterUrlDecode,
	"urlEncode":           ast.FilterUrlEncode,
	"urlQueryParam":       ast.FilterUrlQueryParam,
	"htmlEscape":          ast.FilterHtmlEscape,
	"htmlUnescape":        ast.FilterHtmlUnescape,
	"toHex":               ast.FilterToHex,
	"utf8Decode":          ast.FilterUtf8Decode,
	"utf8Encode":          ast.FilterUtf8Encode,
	"xpath":               ast.FilterXPath,
	"jsonpath":            ast.FilterJSONPath,
	"location":            ast.FilterLocation,
}

func filterNameToType(name string) ast.FilterType {
	if ft, ok := filterNameToTypeMap[name]; ok {
		return ft
	}
	return ast.FilterCount
}

var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT", "QUERY"}

func isMethod(line string) bool {
	for _, m := range httpMethods {
		if strings.HasPrefix(line, m+" ") || line == m {
			return true
		}
	}
	return false
}

func isResponseStatus(line string) bool {
	return strings.HasPrefix(line, "HTTP")
}

func isSection(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

func isBodyStart(line string) bool {
	return strings.HasPrefix(line, "```") ||
		strings.HasPrefix(line, "base64,") ||
		strings.HasPrefix(line, "hex,") ||
		strings.HasPrefix(line, "file,") ||
		(strings.HasPrefix(line, "`") && strings.Count(line, "`") == 2)
}

func looksLikeBody(line string) bool {
	return strings.HasPrefix(line, "{") ||
		strings.HasPrefix(line, "[") ||
		strings.HasPrefix(line, "<?xml")
}
