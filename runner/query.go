package runner

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/wei-lli/hurlx/ast"
	"github.com/wei-lli/hurlx/filter"
	"github.com/wei-lli/hurlx/tmpl"
)

// MaxRegexPatternLen is the maximum allowed length for regex patterns.
const MaxRegexPatternLen = 1024

var cookieAttrRe = regexp.MustCompile(`^(.+?)\[(.+)\]$`)

func (r *Runner) evaluateQuery(query ast.Query, resp *http.Response, body []byte, result *EntryResult, vars tmpl.Variables) (interface{}, error) {
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
		if len(query.Value) > MaxRegexPatternLen {
			return nil, fmt.Errorf("regex: pattern exceeds maximum length of %d", MaxRegexPatternLen)
		}
		re, err := compileRegex(query.Value)
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
	case ast.QueryIP:
		addr := resp.Request.URL.Host
		if host, _, err := net.SplitHostPort(addr); err == nil {
			addr = host
		}
		return addr, nil
	case ast.QueryCertificate:
		if resp.TLS == nil {
			return nil, fmt.Errorf("certificate: no TLS connection")
		}
		if len(resp.TLS.PeerCertificates) == 0 {
			return nil, fmt.Errorf("certificate: no peer certificates")
		}
		cert := resp.TLS.PeerCertificates[0]
		switch query.Value {
		case "Subject":
			return cert.Subject.String(), nil
		case "Issuer":
			return cert.Issuer.String(), nil
		case "SerialNumber":
			return cert.SerialNumber.String(), nil
		case "NotBefore":
			return cert.NotBefore.Format(time.RFC3339), nil
		case "NotAfter":
			return cert.NotAfter.Format(time.RFC3339), nil
		case "DNSNames":
			return cert.DNSNames, nil
		case "EmailAddresses":
			return cert.EmailAddresses, nil
		case "IPAddresses":
			var ips []string
			for _, ip := range cert.IPAddresses {
				ips = append(ips, ip.String())
			}
			return ips, nil
		default:
			return nil, fmt.Errorf("certificate: unknown attribute %q", query.Value)
		}
	case ast.QuerySHA256:
		h := sha256.Sum256(body)
		return hex.EncodeToString(h[:]), nil
	case ast.QueryMD5:
		h := md5.Sum(body)
		return hex.EncodeToString(h[:]), nil
	case ast.QueryVariable:
		if val, ok := vars.Get(query.Value); ok {
			return val, nil
		}
		return nil, fmt.Errorf("variable %q not found", query.Value)
	default:
		return nil, fmt.Errorf("unsupported query type: %d", query.Type)
	}
}

func (r *Runner) extractCookie(resp *http.Response, name string) (interface{}, error) {
	matches := cookieAttrRe.FindStringSubmatch(name)
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
