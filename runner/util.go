package runner

import (
	"compress/gzip"
	"compress/zlib"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	var reader io.Reader = resp.Body
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		reader = gr
	case "deflate":
		var err error
		zr, err := zlib.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		reader = zr
	}
	return io.ReadAll(reader)
}

func resolveFilePath(fileRoot string, path string) string {
	if filepath.IsAbs(path) {
		if fileRoot != "" {
			absFileRoot, _ := filepath.Abs(fileRoot)
			absPath, _ := filepath.Abs(path)
			if !strings.HasPrefix(absPath, absFileRoot+string(filepath.Separator)) && absPath != absFileRoot {
				return ""
			}
		}
		return path
	}
	if fileRoot != "" {
		resolved := filepath.Join(fileRoot, path)
		clean := filepath.Clean(resolved)
		absFileRoot, _ := filepath.Abs(fileRoot)
		if !strings.HasPrefix(clean, absFileRoot+string(filepath.Separator)) && clean != absFileRoot {
			return ""
		}
		return clean
	}
	return path
}

// ParseDuration parses a duration string supporting ms, s, m, h suffixes and
// bare numbers (interpreted as milliseconds). Float values are supported
// (e.g., "1.5s", "0.5m").
func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "ms") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "ms"), 64); err == nil {
			return time.Duration(f * float64(time.Millisecond))
		}
	}
	if strings.HasSuffix(s, "s") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64); err == nil {
			return time.Duration(f * float64(time.Second))
		}
	}
	if strings.HasSuffix(s, "m") && !strings.HasSuffix(s, "ms") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "m"), 64); err == nil {
			return time.Duration(f * float64(time.Minute))
		}
	}
	if strings.HasSuffix(s, "h") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64); err == nil {
			return time.Duration(f * float64(time.Hour))
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(f * float64(time.Millisecond))
	}
	return 0
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

// Type checking helpers for assertions

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

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		time.RFC1123,
		time.RFC1123Z,
		time.RFC850,
		time.ANSIC,
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
