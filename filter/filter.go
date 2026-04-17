package filter

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/antchfx/xmlquery"
	"github.com/wei-lli/hurlx/ast"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

func Apply(value interface{}, filters []ast.Filter) (interface{}, error) {
	var err error
	for _, f := range filters {
		value, err = applyFilter(value, f)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func applyFilter(value interface{}, f ast.Filter) (interface{}, error) {
	switch f.Type {
	case ast.FilterCount:
		return filterCount(value)
	case ast.FilterFirst:
		return filterFirst(value)
	case ast.FilterLast:
		return filterLast(value)
	case ast.FilterNth:
		return filterNth(value, f.Value)
	case ast.FilterToInt:
		return filterToInt(value)
	case ast.FilterToFloat:
		return filterToFloat(value)
	case ast.FilterToString:
		return filterToString(value)
	case ast.FilterSplit:
		return filterSplit(value, f.Value)
	case ast.FilterReplace:
		return filterReplace(value, f.Value)
	case ast.FilterRegex:
		return filterRegex(value, f.Value)
	case ast.FilterReplaceRegex:
		return filterReplaceRegex(value, f.Value)
	case ast.FilterBase64Decode:
		return filterBase64Decode(value)
	case ast.FilterBase64Encode:
		return filterBase64Encode(value)
	case ast.FilterToHex:
		return filterToHex(value)
	case ast.FilterUrlEncode:
		return filterUrlEncode(value)
	case ast.FilterUrlDecode:
		return filterUrlDecode(value)
	case ast.FilterDecode:
		return filterDecode(value, f.Value)
	case ast.FilterToDate:
		return filterToDate(value, f.Value)
	case ast.FilterDateFormat:
		return filterDateFormat(value, f.Value)
	case ast.FilterDaysAfterNow:
		return filterDaysAfterNow(value)
	case ast.FilterDaysBeforeNow:
		return filterDaysBeforeNow(value)
	case ast.FilterLocation:
		return filterLocation(value)
	case ast.FilterXPath:
		return filterXPath(value, f.Value)
	case ast.FilterJSONPath:
		return filterJSONPathFilter(value, f.Value)
	case ast.FilterHtmlEscape:
		return filterHtmlEscape(value)
	case ast.FilterHtmlUnescape:
		return filterHtmlUnescape(value)
	case ast.FilterUtf8Encode:
		return filterUtf8Encode(value)
	case ast.FilterUtf8Decode:
		return filterUtf8Decode(value)
	case ast.FilterUrlQueryParam:
		return filterUrlQueryParam(value, f.Value)
	case ast.FilterBase64UrlSafeDecode:
		return filterBase64UrlSafeDecode(value)
	case ast.FilterBase64UrlSafeEncode:
		return filterBase64UrlSafeEncode(value)
	case ast.FilterUpper:
		return filterUpper(value)
	case ast.FilterLower:
		return filterLower(value)
	default:
		return value, fmt.Errorf("unsupported filter type: %d", f.Type)
	}
}

func filterCount(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		return len(v), nil
	case string:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	case []byte:
		return len(v), nil
	case int:
		return 1, nil
	case int64:
		return 1, nil
	case float64:
		return 1, nil
	default:
		return nil, fmt.Errorf("count: unsupported type %T", value)
	}
}

func filterFirst(value interface{}) (interface{}, error) {
	arr, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("first: expected array, got %T", value)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("first: empty array")
	}
	return arr[0], nil
}

func filterLast(value interface{}) (interface{}, error) {
	arr, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("last: expected array, got %T", value)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("last: empty array")
	}
	return arr[len(arr)-1], nil
}

func filterNth(value interface{}, indexStr string) (interface{}, error) {
	arr, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("nth: expected array, got %T", value)
	}
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fmt.Errorf("nth: invalid index %s", indexStr)
	}
	if idx < 0 {
		idx = len(arr) + idx
	}
	if idx < 0 || idx >= len(arr) {
		return nil, fmt.Errorf("nth: index %d out of range", idx)
	}
	return arr[idx], nil
}

func filterToInt(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("toInt: cannot convert %q", v)
		}
		return n, nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return nil, fmt.Errorf("toInt: unsupported type %T", value)
	}
}

func filterToFloat(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("toFloat: cannot convert %q", v)
		}
		return f, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return nil, fmt.Errorf("toFloat: unsupported type %T", value)
	}
}

func filterToString(value interface{}) (interface{}, error) {
	return fmt.Sprintf("%v", value), nil
}

func filterSplit(value interface{}, delimiter string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("split: expected string, got %T", value)
	}
	parts := strings.Split(str, delimiter)
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result, nil
}

func filterReplace(value interface{}, replaceStr string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("replace: expected string, got %T", value)
	}
	parts := strings.SplitN(replaceStr, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("replace: expected 'old new', got %q", replaceStr)
	}
	return strings.ReplaceAll(str, parts[0], parts[1]), nil
}

const maxRegexPatternLen = 1024

func filterRegex(value interface{}, pattern string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("regex: expected string, got %T", value)
	}
	if len(pattern) > maxRegexPatternLen {
		return nil, fmt.Errorf("regex: pattern exceeds maximum length of %d", maxRegexPatternLen)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("regex: invalid pattern %q: %w", pattern, err)
	}
	matches := re.FindStringSubmatch(str)
	if len(matches) < 2 {
		return nil, fmt.Errorf("regex: no match for %q in %q", pattern, str)
	}
	return matches[1], nil
}

func filterReplaceRegex(value interface{}, replaceStr string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("replaceRegex: expected string, got %T", value)
	}
	parts := strings.SplitN(replaceStr, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("replaceRegex: expected 'pattern replacement', got %q", replaceStr)
	}
	if len(parts[0]) > maxRegexPatternLen {
		return nil, fmt.Errorf("replaceRegex: pattern exceeds maximum length of %d", maxRegexPatternLen)
	}
	re, err := regexp.Compile(parts[0])
	if err != nil {
		return nil, fmt.Errorf("replaceRegex: invalid pattern %q: %w", parts[0], err)
	}
	return re.ReplaceAllString(str, parts[1]), nil
}

func filterBase64Decode(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("base64Decode: expected string, got %T", value)
	}
	decoded, err := decodeBase64(str)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func filterBase64Encode(value interface{}) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("base64Encode: expected bytes, got %T", value)
	}
	return encodeBase64(data), nil
}

func filterToHex(value interface{}) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("toHex: expected bytes, got %T", value)
	}
	return hex.EncodeToString(data), nil
}

func filterUrlEncode(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("urlEncode: expected string, got %T", value)
	}
	return urlEncode(str), nil
}

func filterUrlDecode(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("urlDecode: expected string, got %T", value)
	}
	return urlDecode(str), nil
}

func filterToDate(value interface{}, format string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("toDate: expected string, got %T", value)
	}
	t, err := parseDate(str, format)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func filterDateFormat(value interface{}, format string) (interface{}, error) {
	var t time.Time
	switch v := value.(type) {
	case time.Time:
		t = v
	case string:
		var err error
		t, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, fmt.Errorf("dateFormat: cannot parse date %q: %w", v, err)
		}
	default:
		return nil, fmt.Errorf("dateFormat: unsupported type %T", value)
	}
	return formatDate(t, format), nil
}

func filterDaysAfterNow(value interface{}) (interface{}, error) {
	t, err := toTime(value)
	if err != nil {
		return nil, err
	}
	days := int(time.Until(t).Hours() / 24)
	return days, nil
}

func filterDaysBeforeNow(value interface{}) (interface{}, error) {
	t, err := toTime(value)
	if err != nil {
		return nil, err
	}
	days := int(time.Since(t).Hours() / 24)
	return days, nil
}

func filterLocation(value interface{}) (interface{}, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("location: expected map, got %T", value)
	}
	if loc, ok := m["location"]; ok {
		return loc, nil
	}
	return nil, fmt.Errorf("location: no 'location' key")
}

func filterHtmlEscape(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("htmlEscape: expected string, got %T", value)
	}
	return html.EscapeString(str), nil
}

func filterHtmlUnescape(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("htmlUnescape: expected string, got %T", value)
	}
	return html.UnescapeString(str), nil
}

func filterUtf8Encode(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("utf8Encode: expected string, got %T", value)
	}
	return []byte(str), nil
}

func filterUtf8Decode(value interface{}) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil, fmt.Errorf("utf8Decode: expected bytes, got %T", value)
	}
	return string(data), nil
}

func filterUrlQueryParam(value interface{}, param string) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("urlQueryParam: expected string, got %T", value)
	}
	return getURLQueryParam(str, param)
}

func filterBase64UrlSafeDecode(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("base64UrlSafeDecode: expected string, got %T", value)
	}
	decoded, err := decodeBase64URLSafe(str)
	if err != nil {
		return nil, err
	}
	return string(decoded), nil
}

func filterBase64UrlSafeEncode(value interface{}) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("base64UrlSafeEncode: expected bytes, got %T", value)
	}
	return encodeBase64URLSafe(data), nil
}

func DecodeBase64(s string) ([]byte, error) {
	return decodeBase64(s)
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimRight(s, "=")
	data, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64Decode: %w", err)
	}
	return data, nil
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64URLSafe(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64UrlSafeDecode: %w", err)
	}
	return data, nil
}

func encodeBase64URLSafe(data []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func parseDate(s string, format string) (time.Time, error) {
	goFormat := hurlFormatToGo(format)
	t, err := time.Parse(goFormat, s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(goFormat+" MST", s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(goFormat+" -0700", s)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("toDate: cannot parse %q with format %q", s, format)
}

func formatDate(t time.Time, format string) string {
	goFormat := hurlFormatToGo(format)
	return t.Format(goFormat)
}

func hurlFormatToGo(format string) string {
	if format == "%+" {
		return time.RFC3339
	}
	replacements := []struct {
		strftime string
		goFmt    string
	}{
		{"%Y", "2006"},
		{"%m", "01"},
		{"%d", "02"},
		{"%H", "15"},
		{"%M", "04"},
		{"%S", "05"},
		{"%z", "-0700"},
		{"%Z", "MST"},
		{"%a", "Mon"},
		{"%A", "Monday"},
		{"%b", "Jan"},
		{"%B", "January"},
		{"%I", "03"},
		{"%p", "PM"},
		{"%j", "002"},
		{"%W", "Mon"},
	}
	result := format
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.strftime, r.goFmt)
	}
	return result
}

func toTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, fmt.Errorf("cannot parse time: %w", err)
		}
		return t, nil
	default:
		return time.Time{}, fmt.Errorf("expected time, got %T", value)
	}
}

func urlEncode(s string) string {
	var result strings.Builder
	for _, ch := range s {
		if isUnreserved(ch) || ch == '/' {
			result.WriteRune(ch)
		} else {
			for _, b := range []byte(string(ch)) {
				result.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return result.String()
}

func urlDecode(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '%' && i+2 < len(s) {
			hexStr := s[i+1 : i+3]
			if b, err := strconv.ParseUint(hexStr, 16, 8); err == nil {
				result.WriteByte(byte(b))
				i += 3
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func isUnreserved(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' || ch == '_' || ch == '~'
}

func getURLQueryParam(urlStr string, param string) (string, error) {
	idx := strings.Index(urlStr, "?")
	if idx < 0 {
		return "", fmt.Errorf("no query params in URL")
	}
	query := urlStr[idx+1:]
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && urlDecode(kv[0]) == param {
			return urlDecode(kv[1]), nil
		}
	}
	return "", fmt.Errorf("query param %q not found", param)
}

func filterDecode(value interface{}, encoding string) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil, fmt.Errorf("decode: expected bytes, got %T", value)
	}
	if encoding == "" {
		encoding = "utf-8"
	}
	enc, err := ianaindex.IANA.Encoding(encoding)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("decode: unsupported encoding %q", encoding)
	}
	decoded, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return string(decoded), nil
}

func ExtractJSONPath(data []byte, path string) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("jsonpath: invalid JSON: %w", err)
	}
	return jsonPath(v, path), nil
}

func jsonPath(v interface{}, path string) interface{} {
	path = strings.TrimSpace(path)
	if path == "$" {
		return v
	}

	if strings.HasPrefix(path, "$[") {
		return walkJSON(v, path[1:])
	}
	if strings.HasPrefix(path, "$.") {
		return walkJSON(v, path[2:])
	}

	return walkJSON(v, path)
}

func walkJSON(v interface{}, path string) interface{} {
	if path == "" {
		return v
	}

	if strings.HasPrefix(path, ".'") || strings.HasPrefix(path, ".\"") {
		quote := path[1]
		end := strings.IndexByte(path[2:], quote)
		if end < 0 {
			return nil
		}
		key := path[2 : 2+end]
		rest := path[2+end+1:]
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		return walkJSON(m[key], rest)
	}

	if strings.HasPrefix(path, ".") {
		path = path[1:]
		parts := strings.SplitN(path, ".", 2)
		key := parts[0]
		if idx := strings.Index(key, "["); idx >= 0 {
			key = key[:idx]
		}
		var rest string
		if len(parts) > 1 {
			rest = "." + parts[1]
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		return walkJSON(m[key], rest)
	}

	if strings.HasPrefix(path, "[") {
		end := strings.IndexByte(path, ']')
		if end < 0 {
			return nil
		}
		indexStr := path[1:end]
		rest := path[end+1:]

		if indexStr == "*" {
			arr, ok := v.([]interface{})
			if !ok {
				return nil
			}
			var results []interface{}
			for _, item := range arr {
				results = append(results, walkJSON(item, rest))
			}
			return results
		}

		indexStr = strings.Trim(indexStr, "'\"")
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			m, ok := v.(map[string]interface{})
			if ok {
				return walkJSON(m[indexStr], rest)
			}
			return nil
		}
		arr, ok := v.([]interface{})
		if !ok {
			return nil
		}
		// Handle negative indexes (Python-style)
		if idx < 0 {
			idx = len(arr) + idx
		}
		if idx < 0 || idx >= len(arr) {
			return nil
		}
		return walkJSON(arr[idx], rest)
	}

	if path != "" {
		parts := strings.SplitN(path, ".", 2)
		key := parts[0]
		var rest string
		if len(parts) > 1 {
			rest = parts[1]
		}

		if idx := strings.Index(key, "["); idx >= 0 {
			arrayAccess := key[idx:]
			key = key[:idx]

			m, ok := v.(map[string]interface{})
			if !ok {
				return nil
			}
			arrVal := m[key]
			if rest != "" {
				rest = "." + rest
			}
			return walkJSON(arrVal, arrayAccess+rest)
		}

		m, ok := v.(map[string]interface{})
		if !ok {
			return nil
		}
		return walkJSON(m[key], rest)
	}

	return v
}

func filterXPath(value interface{}, expr string) (interface{}, error) {
	var data string
	switch v := value.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		return nil, fmt.Errorf("xpath: expected string/bytes, got %T", value)
	}

	if strings.Contains(data, "<!DOCTYPE") || strings.Contains(data, "<html") || strings.Contains(data, "<HTML") {
		doc, err := htmlquery.Parse(strings.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("xpath: parse HTML: %w", err)
		}
		node := htmlquery.FindOne(doc, expr)
		if node == nil {
			nodes := htmlquery.Find(doc, expr)
			if len(nodes) == 0 {
				return nil, fmt.Errorf("xpath: no match for %q", expr)
			}
			results := make([]interface{}, len(nodes))
			for i, n := range nodes {
				results[i] = htmlquery.InnerText(n)
			}
			return results, nil
		}
		return htmlquery.InnerText(node), nil
	}

	doc, err := xmlquery.Parse(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("xpath: parse XML: %w", err)
	}
	node := xmlquery.FindOne(doc, expr)
	if node == nil {
		nodes := xmlquery.Find(doc, expr)
		if len(nodes) == 0 {
			return nil, fmt.Errorf("xpath: no match for %q", expr)
		}
		results := make([]interface{}, len(nodes))
		for i, n := range nodes {
			results[i] = n.InnerText()
		}
		return results, nil
	}
	return node.InnerText(), nil
}

func filterJSONPathFilter(value interface{}, expr string) (interface{}, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("jsonpath filter: expected string/bytes, got %T", value)
	}
	return ExtractJSONPath(data, expr)
}

func ExtractXPath(data []byte, expr string, isHTML bool) (interface{}, error) {
	if isHTML {
		doc, err := htmlquery.Parse(strings.NewReader(string(data)))
		if err != nil {
			return nil, fmt.Errorf("xpath: parse HTML: %w", err)
		}
		node := htmlquery.FindOne(doc, expr)
		if node == nil {
			return nil, fmt.Errorf("xpath: no match for %q", expr)
		}
		if node.Type == 2 {
			return node.Data, nil
		}
		return htmlquery.InnerText(node), nil
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("xpath: parse XML: %w", err)
	}
	node := xmlquery.FindOne(doc, expr)
	if node == nil {
		return nil, fmt.Errorf("xpath: no match for %q", expr)
	}
	return node.InnerText(), nil
}

func filterUpper(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("upper: expected string, got %T", value)
	}
	return strings.ToUpper(str), nil
}

func filterLower(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("lower: expected string, got %T", value)
	}
	return strings.ToLower(str), nil
}
