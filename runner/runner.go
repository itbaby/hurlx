package runner

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
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
	Delay           time.Duration
	Retry           int
	RetryInterval   time.Duration
	CACert          string
	Cert            string
	Key             string
	IPv4            bool
	IPv6            bool
}

type RunResult struct {
	Entries []EntryResult
	Success bool
}

type EntryResult struct {
	EntryIndex   int
	Request      *http.Request
	Response     *http.Response
	Body         []byte
	Duration     time.Duration
	Error        error
	Captures     map[string]interface{}
	RedactedVars map[string]bool
}

type Runner struct {
	client           *http.Client
	options          RunOptions
	variables        tmpl.Variables
	logger           *log.Logger
	fileRoot         string
	redirectRecorder *redirectRecorder
	maxRedirects     int
}

type redirectRecorder struct {
	requests []string
}

func NewRunner(opts RunOptions) (*Runner, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		// cookiejar.New only returns an error if the passed options are invalid.
		// Passing nil is always safe, but handle it defensively.
		fmt.Fprintf(os.Stderr, "warning: failed to create cookie jar: %v\n", err)
	}
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

	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.Insecure,
	}

	if opts.CACert != "" {
		caData, err := os.ReadFile(opts.CACert)
		if err != nil {
			return nil, fmt.Errorf("cannot read CA cert %s: %w", opts.CACert, err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("failed to append CA cert from %s", opts.CACert)
		}
		tlsConfig.RootCAs = certPool
	}

	if opts.Cert != "" && opts.Key != "" {
		cert, err := tls.LoadX509KeyPair(opts.Cert, opts.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot load client cert/key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport := &http.Transport{
		TLSClientConfig:    tlsConfig,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	switch opts.HTTPVersion {
	case "3":
		return nil, fmt.Errorf("HTTP/3 is not yet supported")
	case "2":
		transport.ForceAttemptHTTP2 = true
	case "1.1", "1.0":
		transport.ForceAttemptHTTP2 = false
		tlsConfig.NextProtos = []string{"http/1.1"}
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if opts.ConnectTimeout > 0 {
		dialer.Timeout = opts.ConnectTimeout
	}

	if opts.IPv4 {
		transport.DialContext = func(ctx context.Context, _ string, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		}
	} else if opts.IPv6 {
		transport.DialContext = func(ctx context.Context, _ string, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp6", addr)
		}
	} else if opts.ConnectTimeout > 0 {
		transport.DialContext = dialer.DialContext
	}

	if opts.Proxy != "" {
		proxyURL, err := url.Parse(opts.Proxy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid proxy URL %q: %v\n", opts.Proxy, err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	if transport.Proxy == nil {
		transport.Proxy = http.ProxyFromEnvironment
	}

	// TODO: Implement AWS Signature V4 signing when opts.AWSSigV4 is set.

	fileRoot := opts.FileRoot

	r := &Runner{
		options:          opts,
		variables:        variables,
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

	return r, nil
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

		// Apply global delay between entries (not before the first entry)
		if i > start && r.options.Delay > 0 {
			time.Sleep(r.options.Delay)
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
	maxRetries := r.options.Retry
	retryInterval := r.options.RetryInterval
	if retryInterval == 0 {
		retryInterval = time.Second
	}
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
	if isRetry && (r.options.Verbose || r.options.VeryVerbose) {
		r.logger.Printf("* Retrying entry %d\n", index+1)
	}
	result := &EntryResult{
		EntryIndex:   index,
		Captures:     make(map[string]interface{}),
		RedactedVars: make(map[string]bool),
	}

	// Reset redirect recorder for each entry to prevent cross-entry leakage
	r.redirectRecorder.requests = r.redirectRecorder.requests[:0]

	// Clone variables for this entry to avoid polluting shared state
	entryVars := r.variables.Clone()

	if entry.Request.Options != nil && len(entry.Request.Options.Variables) > 0 {
		for k, v := range entry.Request.Options.Variables {
			rendered, err := tmpl.Render(v, entryVars)
			if err == nil {
				entryVars.Set(k, rendered)
			} else {
				entryVars.Set(k, v)
			}
		}
	}

	req, err := r.buildRequest(entry.Request, entryVars)
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

	// Save per-entry client state that may be modified by options, then restore after Do
	savedTimeout := r.client.Timeout
	savedCheckRedirect := r.client.CheckRedirect

	start := time.Now()
	resp, err := r.client.Do(req)
	duration := time.Since(start)
	result.Duration = duration

	r.client.Timeout = savedTimeout
	r.client.CheckRedirect = savedCheckRedirect

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
		if err := r.processResponse(index, entry.Response, result, entryVars); err != nil {
			result.Error = err
			return result, err
		}
	}

	return result, nil
}

func (r *Runner) renderTemplate(templateStr string, vars tmpl.Variables, context string) string {
	rendered, err := tmpl.Render(templateStr, vars)
	if err != nil {
		return templateStr
	}
	if rendered == templateStr && strings.Contains(templateStr, "{{") {
		if r.options.Verbose {
			r.logger.Printf("* warning: unresolved template in %s: %q\n", context, templateStr)
		}
	}
	return rendered
}

func (r *Runner) buildRequest(reqDef *ast.Request, vars tmpl.Variables) (*http.Request, error) {
	method := reqDef.Method

	if r.options.Verbose {
		keys := make([]string, 0)
		for k := range vars {
			keys = append(keys, k)
		}
		r.logger.Printf("* Variables available: %v\n", keys)
	}

	rawURL, err := tmpl.Render(reqDef.URL, vars)
	if err != nil {
		return nil, fmt.Errorf("render URL: %w", err)
	}

	// Normalize URL by escaping spaces
	if strings.Contains(rawURL, " ") {
		if idx := strings.Index(rawURL, "://"); idx >= 0 {
			scheme := rawURL[:idx+3]
			rest := rawURL[idx+3:]
			if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
				authority := rest[:slashIdx]
				pathAndQuery := rest[slashIdx:]
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
			rendered := r.renderTemplate(kv.Value, vars, fmt.Sprintf("query param %s", kv.Key))
			q.Set(kv.Key, rendered)
		}
		parsedURL.RawQuery = q.Encode()
	}

	var bodyBytes []byte
	var contentType string
	if reqDef.Form != nil && len(reqDef.Form) > 0 {
		data := url.Values{}
		for _, kv := range reqDef.Form {
			rendered := r.renderTemplate(kv.Value, vars, fmt.Sprintf("form param %s", kv.Key))
			data.Set(kv.Key, rendered)
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
				if filePath == "" {
					return nil, fmt.Errorf("invalid file path %q: path escapes file root", field.Value)
				}
				fw, err := writer.CreateFormFile(field.Name, filepath.Base(filePath))
				if err != nil {
					return nil, err
				}
				f, err := os.Open(filePath)
				if err != nil {
					return nil, fmt.Errorf("cannot open file %s: %w", filePath, err)
				}
				_, copyErr := io.Copy(fw, f)
				f.Close()
				if copyErr != nil {
					return nil, copyErr
				}
			} else {
				fw, err := writer.CreateFormField(field.Name)
				if err != nil {
					return nil, err
				}
				rendered := r.renderTemplate(field.Value, vars, fmt.Sprintf("multipart field %s", field.Name))
				if _, err := fw.Write([]byte(rendered)); err != nil {
					return nil, err
				}
			}
		}
		writer.Close()
		bodyBytes = buf.Bytes()
		contentType = writer.FormDataContentType()
	} else if reqDef.Body != nil {
		bodyBytes = r.buildBody(reqDef.Body, vars)

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
			if filePath == "" {
				return nil, fmt.Errorf("invalid body file path %q: path escapes file root", reqDef.Body.Content)
			}
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
		rendered := r.renderTemplate(h.Value, vars, fmt.Sprintf("header %s", h.Name))
		req.Header.Set(h.Name, rendered)
	}

	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	if reqDef.Cookies != nil {
		for _, c := range reqDef.Cookies {
			rendered := r.renderTemplate(c.Value, vars, fmt.Sprintf("cookie %s", c.Key))
			req.AddCookie(&http.Cookie{
				Name:  c.Key,
				Value: rendered,
			})
		}
	}

	if reqDef.BasicAuth != nil {
		username := r.renderTemplate(reqDef.BasicAuth.Username, vars, "basic-auth username")
		password := r.renderTemplate(reqDef.BasicAuth.Password, vars, "basic-auth password")
		req.SetBasicAuth(username, password)
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
		req.Header.Set("Accept-Encoding", "gzip, deflate")
	}

	r.applyRequestOptions(req, reqDef.Options, vars)

	return req, nil
}

func (r *Runner) buildBody(body *ast.Body, vars tmpl.Variables) []byte {
	switch body.Type {
	case ast.BodyJSON, ast.BodyXML, ast.BodyMultiline, ast.BodyOneline:
		content, err := tmpl.Render(body.Content, vars)
		if err != nil {
			return []byte(body.Content)
		}
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

func (r *Runner) applyRequestOptions(req *http.Request, opts *ast.OptionsSection, vars tmpl.Variables) {
	if opts == nil {
		return
	}

	// Save state that may be temporarily overridden per-entry
	savedVerbose := r.options.Verbose

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
		rendered, err := tmpl.Render(v, vars)
		if err == nil {
			vars.Set(k, rendered)
			r.variables.Set(k, rendered)
		} else {
			vars.Set(k, v)
			r.variables.Set(k, v)
		}
	}

	for k, v := range opts.Headers {
		rendered := r.renderTemplate(v, vars, fmt.Sprintf("options header %s", k))
		req.Header.Set(k, rendered)
	}

	// Restore state that was temporarily overridden for this entry
	r.options.Verbose = savedVerbose
}

func (r *Runner) processResponse(index int, respDef *ast.Response, result *EntryResult, vars tmpl.Variables) error {
	resp := result.Response
	body := result.Body

	if !r.options.IgnoreAsserts {
		if respDef.Status != 0 && resp.StatusCode != respDef.Status {
			return fmt.Errorf("entry %d: status code assert failed: expected %d, got %d",
				index, respDef.Status, resp.StatusCode)
		}

		for _, hdr := range respDef.Headers {
			expected := r.renderTemplate(hdr.Value, vars, fmt.Sprintf("response header assert %s", hdr.Name))
			actual := resp.Header.Get(hdr.Name)
			if !strings.EqualFold(actual, expected) && actual != expected {
				return fmt.Errorf("entry %d: header assert failed: expected %s=%s, got %s=%s",
					index, hdr.Name, expected, hdr.Name, actual)
			}
		}

		if respDef.Body != nil {
			expectedBody := string(r.buildBody(respDef.Body, vars))
			if string(body) != expectedBody {
				return fmt.Errorf("entry %d: body assert failed\nexpected: %q\ngot: %q", index, expectedBody, string(body))
			}
		}
	}

	for _, cap := range respDef.Captures {
		value, err := r.evaluateQuery(cap.Query, resp, body, result, vars)
		if err != nil {
			return fmt.Errorf("entry %d: capture %s failed: %w", index, cap.Variable, err)
		}

		if len(cap.Filters) > 0 {
			value, err = filter.Apply(value, cap.Filters)
			if err != nil {
				return fmt.Errorf("entry %d: capture %s filter failed: %w", index, cap.Variable, err)
			}
		}

		vars.Set(cap.Variable, value)
		r.variables.Set(cap.Variable, value)

		if cap.Redact {
			result.RedactedVars[cap.Variable] = true
			result.Captures[cap.Variable] = "[REDACTED]"
		} else {
			result.Captures[cap.Variable] = value
		}
	}

	for _, assert := range respDef.Asserts {
		value, err := r.evaluateQuery(assert.Query, resp, body, result, vars)
		if err != nil {
			if assert.Predicate == ast.PredExists {
				r.checkAssert(index, assert, nil, false, vars)
				continue
			}
			return fmt.Errorf("entry %d: assert query failed: %w", index, err)
		}

		exists := value != nil
		if str, ok := value.(string); ok && str == "" {
			exists = false
		}
		if assert.Predicate == ast.PredExists {
			if err := r.checkAssert(index, assert, value, exists, vars); err != nil {
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

		if err := r.checkAssert(index, assert, value, true, vars); err != nil {
			if !r.options.ContinueOnError {
				return err
			}
		}
	}

	return nil
}
