package runner

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wei-lli/hurlx/ast"
	"github.com/wei-lli/hurlx/filter"
	"github.com/wei-lli/hurlx/tmpl"
)

type RunOptions struct {
	Variables       tmpl.Variables
	Insecure        bool
	FollowRedirect  bool
	MaxRedirects    int
	Timeout         time.Duration
	ConnectTimeout  time.Duration
	Compressed      bool
	Verbose         bool
	VeryVerbose     bool
	Include         bool
	IgnoreAsserts   bool
	ContinueOnError bool
	FromEntry       int
	ToEntry         int
	Output          string
	FileRoot        string
	Proxy           string
	HTTPVersion     string
	User            string
	UserAgent       string
	Trace           bool
}

type RunResult struct {
	Entries []EntryResult
	Success bool
}

type EntryResult struct {
	EntryIndex int
	Request    *http.Request
	Response   *http.Response
	Body       []byte
	Duration   time.Duration
	Error      error
	Captures   map[string]interface{}
}

type Runner struct {
	client           *http.Client
	options          RunOptions
	variables        tmpl.Variables
	cookies          http.CookieJar
	logger           *log.Logger
	fileRoot         string
	redirectRecorder *redirectRecorder
	maxRedirects     int
}

type redirectRecorder struct {
	requests []string
}

func NewRunner(opts RunOptions) *Runner {
	jar, _ := cookiejar.New(nil)
	variables := opts.Variables
	if variables == nil {
		variables = tmpl.NewVariables()
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	maxRedirects := opts.MaxRedirects
	if maxRedirects == 0 {
		maxRedirects = 50
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.Insecure,
		},
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	fileRoot := opts.FileRoot

	r := &Runner{
		options:          opts,
		variables:        variables,
		cookies:          jar,
		logger:           log.New(os.Stderr, "", 0),
		fileRoot:         fileRoot,
		redirectRecorder: &redirectRecorder{},
		maxRedirects:     maxRedirects,
	}

	r.client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			r.redirectRecorder.requests = append(r.redirectRecorder.requests, req.URL.String())
			if len(via) >= r.maxRedirects {
				return fmt.Errorf("stopped after %d redirects", r.maxRedirects)
			}
			if !opts.FollowRedirect {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return r
}

func (r *Runner) Run(entries []ast.Entry) (*RunResult, error) {
	result := &RunResult{
		Success: true,
	}

	start := optsFromEntry(r.options.FromEntry)
	end := optsToEntry(r.options.ToEntry, len(entries))

	for i := start; i < end; i++ {
		entry := entries[i]
		if entry.Request == nil {
			continue
		}

		if entry.Request.Options != nil && entry.Request.Options.Skip != nil && *entry.Request.Options.Skip {
			continue
		}

		entryResult, err := r.runEntry(i, entry)
		if err != nil {
			result.Success = false
			entryResult = &EntryResult{
				EntryIndex: i,
				Error:      err,
			}
		}

		result.Entries = append(result.Entries, *entryResult)

		if r.options.Trace {
			r.traceEntry(entryResult)
		}

		if err != nil {
			if !r.options.ContinueOnError {
				return result, err
			}
		}

		if entryResult.Error != nil && !r.options.ContinueOnError {
			break
		}
	}

	return result, nil
}

func (r *Runner) traceEntry(e *EntryResult) {
	trace := map[string]interface{}{
		"entry":    e.EntryIndex + 1,
		"duration": int64(e.Duration / time.Millisecond),
	}

	if e.Request != nil {
		trace["method"] = e.Request.Method
		trace["url"] = e.Request.URL.String()
	}

	if e.Response != nil {
		trace["status"] = e.Response.StatusCode
	}

	if e.Body != nil {
		trace["body"] = string(e.Body)
	}

	if e.Error != nil {
		trace["error"] = e.Error.Error()
	}

	if len(e.Captures) > 0 {
		trace["captures"] = e.Captures
	}

	data, _ := json.MarshalIndent(trace, "", "  ")
	r.logger.Printf("[trace] %s\n", string(data))
}

func (r *Runner) runEntry(index int, entry ast.Entry) (*EntryResult, error) {
	maxRetries := 0
	retryInterval := time.Second
	if entry.Request.Options != nil {
		if entry.Request.Options.Retry != nil && *entry.Request.Options.Retry > 0 {
			maxRetries = *entry.Request.Options.Retry
		}
		if entry.Request.Options.RetryInterval != "" {
			if d := ParseDuration(entry.Request.Options.RetryInterval); d > 0 {
				retryInterval = d
			}
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := r.executeEntry(index, entry, attempt > 0)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt < maxRetries {
			time.Sleep(retryInterval)
		}
	}
	return nil, lastErr
}

func (r *Runner) executeEntry(index int, entry ast.Entry, isRetry bool) (*EntryResult, error) {
	result := &EntryResult{
		EntryIndex: index,
		Captures:   make(map[string]interface{}),
	}

	if entry.Request.Options != nil && len(entry.Request.Options.Variables) > 0 {
		for k, v := range entry.Request.Options.Variables {
			r.variables.Set(k, tmpl.Render(v, r.variables))
		}
	}

	req, err := r.buildRequest(entry.Request)
	if err != nil {
		return result, fmt.Errorf("entry %d: build request failed: %w", index, err)
	}

	if r.options.Verbose || r.options.VeryVerbose {
		r.logger.Printf("> %s %s\n", req.Method, req.URL.String())
		for k, v := range req.Header {
			r.logger.Printf("> %s: %s\n", k, strings.Join(v, ", "))
		}
	}

	if entry.Request.Options != nil && entry.Request.Options.Delay != "" {
		delay := ParseDuration(entry.Request.Options.Delay)
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	duration := time.Since(start)
	result.Duration = duration

	if err != nil {
		return result, fmt.Errorf("entry %d: request failed: %w", index, err)
	}
	result.Request = req
	result.Response = resp

	body, err := readBody(resp)
	if err != nil {
		return result, fmt.Errorf("entry %d: read body failed: %w", index, err)
	}
	result.Body = body

	if r.options.Verbose || r.options.VeryVerbose {
		r.logger.Printf("< %s %d\n", resp.Proto, resp.StatusCode)
		for k, v := range resp.Header {
			r.logger.Printf("< %s: %s\n", k, strings.Join(v, ", "))
		}
		if r.options.VeryVerbose {
			r.logger.Printf("< Body (%d bytes):\n%s\n", len(body), string(body))
		}
		r.logger.Printf("* Duration: %s\n", duration)
	}

	if entry.Response != nil {
		if err := r.processResponse(index, entry.Response, result); err != nil {
			result.Error = err
			return result, err
		}
	}

	return result, nil
}

func (r *Runner) buildRequest(reqDef *ast.Request) (*http.Request, error) {
	method := reqDef.Method

	if r.options.Verbose {
		keys := make([]string, 0)
		for k := range r.variables {
			keys = append(keys, k)
		}
		r.logger.Printf("* Variables available: %v\n", keys)
	}

	rawURL := tmpl.Render(reqDef.URL, r.variables)

	// Normalize URL by escaping spaces
	if strings.Contains(rawURL, " ") {
		// Find the scheme and authority, then encode the path and query
		if idx := strings.Index(rawURL, "://"); idx >= 0 {
			scheme := rawURL[:idx+3]
			rest := rawURL[idx+3:]
			if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
				authority := rest[:slashIdx]
				pathAndQuery := rest[slashIdx:]
				// Replace spaces with %20 in path and query
				pathAndQuery = strings.ReplaceAll(pathAndQuery, " ", "%20")
				rawURL = scheme + authority + pathAndQuery
			} else if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
				authority := rest[:qIdx]
				query := rest[qIdx:]
				query = strings.ReplaceAll(query, " ", "%20")
				rawURL = scheme + authority + query
			}
		}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be http or https, got %q", parsedURL.Scheme)
	}

	if len(reqDef.Query) > 0 {
		q := parsedURL.Query()
		for _, kv := range reqDef.Query {
			q.Set(kv.Key, tmpl.Render(kv.Value, r.variables))
		}
		parsedURL.RawQuery = q.Encode()
	}

	var bodyBytes []byte
	var contentType string
	if reqDef.Form != nil && len(reqDef.Form) > 0 {
		data := url.Values{}
		for _, kv := range reqDef.Form {
			data.Set(kv.Key, tmpl.Render(kv.Value, r.variables))
		}
		encoded := data.Encode()
		bodyBytes = []byte(encoded)
		contentType = "application/x-www-form-urlencoded"
	} else if reqDef.Multipart != nil && len(reqDef.Multipart) > 0 {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for _, field := range reqDef.Multipart {
			if field.IsFile {
				filePath := resolveFilePath(r.fileRoot, field.Value)
				fw, err := writer.CreateFormFile(field.Name, filepath.Base(filePath))
				if err != nil {
					return nil, err
				}
				f, err := os.Open(filePath)
				if err != nil {
					return nil, fmt.Errorf("cannot open file %s: %w", filePath, err)
				}
				defer f.Close()
				if _, err := io.Copy(fw, f); err != nil {
					return nil, err
				}
			} else {
				fw, err := writer.CreateFormField(field.Name)
				if err != nil {
					return nil, err
				}
				_, err = fw.Write([]byte(tmpl.Render(field.Value, r.variables)))
				if err != nil {
					return nil, err
				}
			}
		}
		writer.Close()
		bodyBytes = buf.Bytes()
		contentType = writer.FormDataContentType()
	} else if reqDef.Body != nil {
		bodyBytes = r.buildBody(reqDef.Body)

		switch reqDef.Body.Type {
		case ast.BodyJSON:
			if contentType == "" {
				contentType = "application/json"
			}
		case ast.BodyXML:
			if contentType == "" {
				contentType = "application/xml"
			}
		case ast.BodyFile:
			filePath := resolveFilePath(r.fileRoot, reqDef.Body.Content)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("cannot read body file %s: %w", filePath, err)
			}
			bodyBytes = data
		}
	}

	req := &http.Request{
		Method:        method,
		URL:           parsedURL,
		Header:        make(http.Header),
		ContentLength: int64(len(bodyBytes)),
		GetBody: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		},
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	for _, h := range reqDef.Headers {
		value := tmpl.Render(h.Value, r.variables)
		req.Header.Set(h.Name, value)
	}

	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	if reqDef.Cookies != nil {
		for _, c := range reqDef.Cookies {
			req.AddCookie(&http.Cookie{
				Name:  c.Key,
				Value: tmpl.Render(c.Value, r.variables),
			})
		}
	}

	if reqDef.BasicAuth != nil {
		req.SetBasicAuth(
			tmpl.Render(reqDef.BasicAuth.Username, r.variables),
			tmpl.Render(reqDef.BasicAuth.Password, r.variables),
		)
	}

	if r.options.User != "" {
		parts := strings.SplitN(r.options.User, ":", 2)
		if len(parts) == 2 {
			req.SetBasicAuth(parts[0], parts[1])
		}
	}

	if r.options.UserAgent != "" {
		req.Header.Set("User-Agent", r.options.UserAgent)
	} else {
		req.Header.Set("User-Agent", "hurlx/1.0")
	}

	if r.options.Compressed {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}

	r.applyRequestOptions(req, reqDef.Options)

	return req, nil
}

func (r *Runner) buildBody(body *ast.Body) []byte {
	switch body.Type {
	case ast.BodyJSON, ast.BodyXML, ast.BodyMultiline, ast.BodyOneline:
		content := tmpl.Render(body.Content, r.variables)
		return []byte(content)
	case ast.BodyBase64:
		decoded, err := filter.DecodeBase64(body.Content)
		if err != nil {
			return []byte(body.Content)
		}
		return decoded
	case ast.BodyHex:
		decoded, err := hex.DecodeString(body.Content)
		if err != nil {
			return []byte(body.Content)
		}
		return decoded
	default:
		return []byte(body.Content)
	}
}

func (r *Runner) applyRequestOptions(req *http.Request, opts *ast.OptionsSection) {
	if opts == nil {
		return
	}

	if opts.Location != nil && *opts.Location {
		maxRedirs := r.maxRedirects
		r.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			r.redirectRecorder.requests = append(r.redirectRecorder.requests, req.URL.String())
			if len(via) >= maxRedirs {
				return fmt.Errorf("stopped after %d redirects", maxRedirs)
			}
			return nil
		}
	}

	if opts.Verbose != nil && *opts.Verbose {
		r.options.Verbose = true
	}

	if opts.Timeout != "" {
		if d := ParseDuration(opts.Timeout); d > 0 {
			r.client.Timeout = d
		}
	}

	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	for k, v := range opts.Variables {
		r.variables.Set(k, tmpl.Render(v, r.variables))
	}
	
	for k, v := range opts.Headers {
		req.Header.Set(k, tmpl.Render(v, r.variables))
	}
}

func (r *Runner) processResponse(index int, respDef *ast.Response, result *EntryResult) error {
	resp := result.Response
	body := result.Body

	if !r.options.IgnoreAsserts {
		if respDef.Status != 0 && resp.StatusCode != respDef.Status {
			return fmt.Errorf("entry %d: status code assert failed: expected %d, got %d",
				index, respDef.Status, resp.StatusCode)
		}

		for _, hdr := range respDef.Headers {
			expected := tmpl.Render(hdr.Value, r.variables)
			actual := resp.Header.Get(hdr.Name)
			if !strings.EqualFold(actual, expected) && actual != expected {
				return fmt.Errorf("entry %d: header assert failed: expected %s=%s, got %s=%s",
					index, hdr.Name, expected, hdr.Name, actual)
			}
		}

		if respDef.Body != nil {
			expectedBody := string(r.buildBody(respDef.Body))
			if string(body) != expectedBody {
				return fmt.Errorf("entry %d: body assert failed", index)
			}
		}
	}

	for _, cap := range respDef.Captures {
		value, err := r.evaluateQuery(cap.Query, resp, body, result)
		if err != nil {
			return fmt.Errorf("entry %d: capture %s failed: %w", index, cap.Variable, err)
		}

		if len(cap.Filters) > 0 {
			value, err = filter.Apply(value, cap.Filters)
			if err != nil {
				return fmt.Errorf("entry %d: capture %s filter failed: %w", index, cap.Variable, err)
			}
		}

		r.variables.Set(cap.Variable, value)
		result.Captures[cap.Variable] = value
	}

	for _, assert := range respDef.Asserts {
		value, err := r.evaluateQuery(assert.Query, resp, body, result)
		if err != nil {
			if assert.Predicate == ast.PredExists {
				r.checkAssert(index, assert, nil, false)
				continue
			}
			return fmt.Errorf("entry %d: assert query failed: %w", index, err)
		}

		exists := value != nil
		if str, ok := value.(string); ok && str == "" {
			exists = false
		}
		if assert.Predicate == ast.PredExists {
			if err := r.checkAssert(index, assert, value, exists); err != nil {
				if !r.options.ContinueOnError {
					return err
				}
			}
			continue
		}

		if len(assert.Filters) > 0 {
			value, err = filter.Apply(value, assert.Filters)
			if err != nil {
				return fmt.Errorf("entry %d: assert filter failed: %w", index, err)
			}
		}

		if err := r.checkAssert(index, assert, value, true); err != nil {
			if !r.options.ContinueOnError {
				return err
			}
		}
	}

	return nil
}

func (r *Runner) evaluateQuery(query ast.Query, resp *http.Response, body []byte, result *EntryResult) (interface{}, error) {
	switch query.Type {
	case ast.QueryStatus:
		return resp.StatusCode, nil
	case ast.QueryVersion:
		return strings.TrimPrefix(resp.Proto, "HTTP/"), nil
	case ast.QueryHeader:
		return resp.Header.Get(query.Value), nil
	case ast.QueryBody:
		return string(body), nil
	case ast.QueryBytes:
		return body, nil
	case ast.QueryJSONPath:
		return filter.ExtractJSONPath(body, query.Value)
	case ast.QueryXPath:
		isHTML := strings.Contains(resp.Header.Get("Content-Type"), "html") ||
			strings.HasPrefix(string(body), "<!doctype") ||
			strings.HasPrefix(string(body), "<!DOCTYPE") ||
			strings.HasPrefix(string(body), "<html") ||
			strings.HasPrefix(string(body), "<HTML")
		return filter.ExtractXPath(body, query.Value, isHTML)
	case ast.QueryRegex:
		if len(query.Value) > maxRegexPatternLen {
			return nil, fmt.Errorf("regex: pattern exceeds maximum length of %d", maxRegexPatternLen)
		}
		re, err := regexp.Compile(query.Value)
		if err != nil {
			return nil, fmt.Errorf("regex: invalid pattern %q: %w", query.Value, err)
		}
		matches := re.FindStringSubmatch(string(body))
		if len(matches) < 2 {
			return nil, fmt.Errorf("regex: no match for %q", query.Value)
		}
		return matches[1], nil
	case ast.QueryDuration:
		return int64(result.Duration / time.Millisecond), nil
	case ast.QueryURL:
		return resp.Request.URL.String(), nil
	case ast.QueryRedirects:
		redirects := make([]interface{}, len(r.redirectRecorder.requests))
		for i, url := range r.redirectRecorder.requests {
			redirects[i] = map[string]interface{}{
				"location": url,
			}
		}
		return redirects, nil
	case ast.QueryCookie:
		return r.extractCookie(resp, query.Value)
	case ast.QuerySHA256:
		h := sha256.Sum256(body)
		return hex.EncodeToString(h[:]), nil
	case ast.QueryMD5:
		h := md5.Sum(body)
		return hex.EncodeToString(h[:]), nil
	case ast.QueryVariable:
		if val, ok := r.variables.Get(query.Value); ok {
			return val, nil
		}
		return nil, fmt.Errorf("variable %q not found", query.Value)
	default:
		return nil, fmt.Errorf("unsupported query type: %d", query.Type)
	}
}

func (r *Runner) extractCookie(resp *http.Response, name string) (interface{}, error) {
	attrRe := regexp.MustCompile(`^(.+?)\[(.+)\]$`)
	matches := attrRe.FindStringSubmatch(name)
	if len(matches) == 3 {
		cookieName := matches[1]
		attrName := matches[2]
		for _, c := range resp.Cookies() {
			if c.Name == cookieName {
				switch attrName {
				case "Value":
					return c.Value, nil
				case "Expires":
					return c.Expires.Format(time.RFC1123), nil
				case "Max-Age":
					return c.MaxAge, nil
				case "Domain":
					return c.Domain, nil
				case "Path":
					return c.Path, nil
				case "Secure":
					return c.Secure, nil
				case "HttpOnly":
					return c.HttpOnly, nil
				case "SameSite":
					return int(c.SameSite), nil
				default:
					return nil, fmt.Errorf("cookie %q: unknown attribute %q", cookieName, attrName)
				}
			}
		}
		return nil, fmt.Errorf("cookie %q not found", cookieName)
	}
	cookies := resp.Cookies()
	for _, c := range cookies {
		if c.Name == name {
			return c.Value, nil
		}
	}
	return "", fmt.Errorf("cookie %q not found", name)
}

func (r *Runner) checkAssert(index int, assert ast.Assert, value interface{}, exists bool) error {
	if assert.Predicate == ast.PredExists {
		if assert.Not {
			if exists {
				return fmt.Errorf("entry %d: assert failed: expected not exists", index)
			}
			return nil
		}
		if !exists {
			return fmt.Errorf("entry %d: assert failed: expected exists", index)
		}
		return nil
	}

	typeChecks := map[ast.PredicateType]func(interface{}) bool{
		ast.PredIsString:     isString,
		ast.PredIsNumber:     isNumber,
		ast.PredIsInteger:    isInteger,
		ast.PredIsFloat:      isFloat,
		ast.PredIsBoolean:    isBool,
		ast.PredIsList:       isList,
		ast.PredIsObject:     isObject,
		ast.PredIsEmpty:      isEmpty,
		ast.PredIsIpv4:       isIPv4,
		ast.PredIsIpv6:       isIPv6,
		ast.PredIsIsoDate:    isISODate,
		ast.PredIsUuid:       isUUID,
		ast.PredIsCollection: isCollection,
		ast.PredIsDate:       isDate,
	}

	if checkFn, ok := typeChecks[assert.Predicate]; ok {
		result := checkFn(value)
		if assert.Not {
			result = !result
		}
		if !result {
			return fmt.Errorf("entry %d: type assert failed for value %v", index, value)
		}
		return nil
	}

	assertVal := assert.Value
	if assertVal.Type == ast.ValueString {
		if strings.Contains(assertVal.Str, "{{") {
			rendered := tmpl.Render(assertVal.Str, r.variables)
			if rendered != assertVal.Str {
				assertVal.Str = rendered
				if intVal, err := strconv.ParseInt(rendered, 10, 64); err == nil {
					assertVal.Type = ast.ValueInt
					assertVal.Int = intVal
				} else if floatVal, err := strconv.ParseFloat(rendered, 64); err == nil {
					assertVal.Type = ast.ValueFloat
					assertVal.Float = floatVal
				}
			}
		}
	}

	cmpResult := compareValues(value, assertVal)

	switch assert.Predicate {
	case ast.PredEqual:
		if assert.Not {
			if cmpResult == 0 {
				return fmt.Errorf("entry %d: assert failed: %v should not equal %v", index, value, formatAssertValue(assertVal))
			}
			return nil
		}
		if cmpResult != 0 {
			return fmt.Errorf("entry %d: assert failed: expected %v, got %v", index, formatAssertValue(assertVal), value)
		}
	case ast.PredNotEqual:
		if assert.Not {
			if cmpResult != 0 {
				return fmt.Errorf("entry %d: assert failed: expected equal, got different", index)
			}
			return nil
		}
		if cmpResult == 0 {
			return fmt.Errorf("entry %d: assert failed: expected different, got equal", index)
		}
	case ast.PredGreaterThan:
		if cmpResult <= 0 {
			return fmt.Errorf("entry %d: assert failed: %v not greater than %v", index, value, formatAssertValue(assertVal))
		}
	case ast.PredGreaterEqual:
		if cmpResult < 0 {
			return fmt.Errorf("entry %d: assert failed: %v not >= %v", index, value, formatAssertValue(assertVal))
		}
	case ast.PredLessThan:
		if cmpResult >= 0 {
			return fmt.Errorf("entry %d: assert failed: %v not less than %v", index, value, formatAssertValue(assertVal))
		}
	case ast.PredLessEqual:
		if cmpResult > 0 {
			return fmt.Errorf("entry %d: assert failed: %v not <= %v", index, value, formatAssertValue(assertVal))
		}
	case ast.PredContains:
		err := checkContains(value, assertVal, assert.Not)
		if err != nil {
			return fmt.Errorf("entry %d: %w", index, err)
		}
	case ast.PredIncludes:
		err := checkIncludes(value, assertVal, assert.Not)
		if err != nil {
			return fmt.Errorf("entry %d: %w", index, err)
		}
	case ast.PredStartsWith:
		err := checkStartsWith(value, assertVal, assert.Not)
		if err != nil {
			return fmt.Errorf("entry %d: %w", index, err)
		}
	case ast.PredEndsWith:
		err := checkEndsWith(value, assertVal, assert.Not)
		if err != nil {
			return fmt.Errorf("entry %d: %w", index, err)
		}
	case ast.PredMatches:
		err := checkMatches(value, assertVal, assert.Not)
		if err != nil {
			return fmt.Errorf("entry %d: %w", index, err)
		}
	}

	return nil
}

func readBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser = resp.Body
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
	case "deflate":
		var err error
		reader, err = zlib.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
	}
	return io.ReadAll(reader)
}

func resolveFilePath(fileRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if fileRoot != "" {
		return filepath.Join(fileRoot, path)
	}
	return path
}

func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "ms") {
		if n, err := strconv.ParseInt(strings.TrimSuffix(s, "ms"), 10, 64); err == nil {
			return time.Duration(n) * time.Millisecond
		}
	}
	if strings.HasSuffix(s, "s") {
		if n, err := strconv.ParseInt(strings.TrimSuffix(s, "s"), 10, 64); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	if strings.HasSuffix(s, "m") && !strings.HasSuffix(s, "ms") {
		if n, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64); err == nil {
			return time.Duration(n) * time.Minute
		}
	}
	if strings.HasSuffix(s, "h") {
		if n, err := strconv.ParseInt(strings.TrimSuffix(s, "h"), 10, 64); err == nil {
			return time.Duration(n) * time.Hour
		}
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(n) * time.Millisecond
	}
	return 0
}

func compareValues(actual interface{}, expected ast.AssertValue) int {
	switch expected.Type {
	case ast.ValueString:
		actualStr := fmt.Sprintf("%v", actual)
		if actualStr < expected.Str {
			return -1
		}
		if actualStr > expected.Str {
			return 1
		}
		return 0
	case ast.ValueInt:
		actualNum := toFloat64(actual)
		if actualNum < float64(expected.Int) {
			return -1
		}
		if actualNum > float64(expected.Int) {
			return 1
		}
		return 0
	case ast.ValueFloat:
		actualNum := toFloat64(actual)
		if actualNum < expected.Float {
			return -1
		}
		if actualNum > expected.Float {
			return 1
		}
		return 0
	case ast.ValueBool:
		actualBool := toBool(actual)
		if actualBool == expected.Bool {
			return 0
		}
		return 1
	case ast.ValueNull:
		if actual == nil {
			return 0
		}
		return 1
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		return 0
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true"
	default:
		return false
	}
}

func formatAssertValue(v ast.AssertValue) string {
	switch v.Type {
	case ast.ValueString:
		return v.Str
	case ast.ValueInt:
		return strconv.FormatInt(v.Int, 10)
	case ast.ValueFloat:
		return strconv.FormatFloat(v.Float, 'f', -1, 64)
	case ast.ValueBool:
		return strconv.FormatBool(v.Bool)
	case ast.ValueNull:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func checkContains(value interface{}, expected ast.AssertValue, not bool) error {
	actual := fmt.Sprintf("%v", value)
	needle := expected.Str
	contains := strings.Contains(actual, needle)
	if not {
		if contains {
			return fmt.Errorf("expected not to contain %q", needle)
		}
		return nil
	}
	if !contains {
		return fmt.Errorf("expected to contain %q, got %q", needle, actual)
	}
	return nil
}

func checkIncludes(value interface{}, expected ast.AssertValue, not bool) error {
	var collection []interface{}
	switch v := value.(type) {
	case []interface{}:
		collection = v
	case []string:
		for _, s := range v {
			collection = append(collection, s)
		}
	case []int:
		for _, i := range v {
			collection = append(collection, i)
		}
	case []int64:
		for _, i := range v {
			collection = append(collection, i)
		}
	case []float64:
		for _, f := range v {
			collection = append(collection, f)
		}
	default:
		return fmt.Errorf("includes: expected collection, got %T", value)
	}

	var found bool
	for _, item := range collection {
		if fmt.Sprintf("%v", item) == expected.Str {
			found = true
			break
		}
		if expected.Type == ast.ValueInt {
			if i, ok := item.(int64); ok && i == expected.Int {
				found = true
				break
			}
			if i, ok := item.(int); ok && int64(i) == expected.Int {
				found = true
				break
			}
		}
		if expected.Type == ast.ValueFloat {
			if f, ok := item.(float64); ok && f == expected.Float {
				found = true
				break
			}
		}
	}

	if not {
		if found {
			return fmt.Errorf("expected collection to not include %v", expected.Str)
		}
		return nil
	}
	if !found {
		return fmt.Errorf("expected collection to include %v", expected.Str)
	}
	return nil
}

func checkStartsWith(value interface{}, expected ast.AssertValue, not bool) error {
	actual := fmt.Sprintf("%v", value)
	prefix := expected.Str
	startsWith := strings.HasPrefix(actual, prefix)
	if not {
		if startsWith {
			return fmt.Errorf("expected not to start with %q", prefix)
		}
		return nil
	}
	if !startsWith {
		return fmt.Errorf("expected to start with %q, got %q", prefix, actual)
	}
	return nil
}

func checkEndsWith(value interface{}, expected ast.AssertValue, not bool) error {
	actual := fmt.Sprintf("%v", value)
	suffix := expected.Str
	endsWith := strings.HasSuffix(actual, suffix)
	if not {
		if endsWith {
			return fmt.Errorf("expected not to end with %q", suffix)
		}
		return nil
	}
	if !endsWith {
		return fmt.Errorf("expected to end with %q, got %q", suffix, actual)
	}
	return nil
}

func checkMatches(value interface{}, expected ast.AssertValue, not bool) error {
	actual := fmt.Sprintf("%v", value)
	pattern := expected.Str
	matched, err := regexpMatch(pattern, actual)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	if not {
		if matched {
			return fmt.Errorf("expected not to match %q", pattern)
		}
		return nil
	}
	if !matched {
		return fmt.Errorf("expected to match %q, got %q", pattern, actual)
	}
	return nil
}

const maxRegexPatternLen = 1024

func regexpMatch(pattern string, s string) (bool, error) {
	if len(pattern) > maxRegexPatternLen {
		return false, fmt.Errorf("regex: pattern exceeds maximum length of %d", maxRegexPatternLen)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}

func isString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

func isNumber(v interface{}) bool {
	return isInteger(v) || isFloat(v)
}

func isInteger(v interface{}) bool {
	switch v.(type) {
	case int, int64, int32:
		return true
	default:
		return false
	}
}

func isFloat(v interface{}) bool {
	switch v.(type) {
	case float64, float32:
		return true
	default:
		return false
	}
}

func isBool(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}

func isList(v interface{}) bool {
	_, ok := v.([]interface{})
	return ok
}

func isObject(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

func isEmpty(v interface{}) bool {
	switch val := v.(type) {
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case string:
		return val == ""
	default:
		return false
	}
}

func isIPv4(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

func isIPv6(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}

func isISODate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	
	// Try multiple ISO 8601 and common HTTP date formats
	formats := []string{
		time.RFC3339,         // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,     // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02",         // Basic ISO 8601 date
		"2006-01-02T15:04:05", // ISO 8601 without timezone
		time.RFC1123,         // Mon, 02 Jan 2006 15:04:05 MST (HTTP date format)
		time.RFC1123Z,        // Mon, 02 Jan 2006 15:04:05 -0700
		time.RFC822,          // 02 Jan 06 15:04 MST
		time.RFC822Z,         // 02 Jan 06 15:04 -0700
		"2006-01-02 15:04:05",  // Common format
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, s); err == nil {
			return true
		}
	}
	return false
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return uuidRegex.MatchString(s)
}

func isCollection(v interface{}) bool {
	switch v.(type) {
	case []interface{}, map[string]interface{}:
		return true
	default:
		return false
	}
}

func isDate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, f := range formats {
		if _, err := time.Parse(f, s); err == nil {
			return true
		}
	}
	return false
}

func optsFromEntry(n int) int {
	if n <= 0 {
		return 0
	}
	return n - 1
}

func optsToEntry(n int, total int) int {
	if n <= 0 || n > total {
		return total
	}
	return n
}
