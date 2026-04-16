package tmpl

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var templateRegex = regexp.MustCompile(`\{\{(.+?)\}\}`)

type Variables map[string]interface{}

func NewVariables() Variables {
	return make(Variables)
}

func (v Variables) Clone() Variables {
	clone := make(Variables)
	for k, val := range v {
		clone[k] = val
	}
	return clone
}

func (v Variables) Set(name string, value interface{}) {
	v[name] = value
}

func (v Variables) Get(name string) (interface{}, bool) {
	val, ok := v[name]
	return val, ok
}

func Render(input string, vars Variables) (string, error) {
	var renderErr error
	result := templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		if renderErr != nil {
			return match
		}
		expr := strings.TrimSpace(match[2 : len(match)-2])
		val, err := evaluateExpr(expr, vars)
		if err != nil {
			renderErr = err
			return match
		}
		return fmt.Sprintf("%v", val)
	})
	return result, renderErr
}

func evaluateExpr(expr string, vars Variables) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	switch {
	case expr == "newUuid" || expr == "uuid":
		val, err := generateUUID()
		return val, err
	case expr == "newDate":
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	case strings.HasPrefix(expr, "date"):
		arg := strings.TrimSpace(strings.TrimPrefix(expr, "date"))
		arg = strings.Trim(arg, " \"'")
		if arg == "" {
			return time.Now().UTC().Format(time.RFC3339Nano), nil
		}
		goFormat := hurlFormatToGo(arg)
		return time.Now().UTC().Format(goFormat), nil
	case strings.HasPrefix(expr, "randomHex"):
		arg := strings.TrimSpace(strings.TrimPrefix(expr, "randomHex"))
		arg = strings.Trim(arg, " \"'")
		n := 32
		if arg != "" {
			if parsed, err := fmt.Sscanf(arg, "%d", &n); err != nil || parsed == 0 {
				n = 32
			}
		}
		return generateRandomHex(n)
	case strings.HasPrefix(expr, "getEnv") || strings.HasPrefix(expr, "getenv"):
		prefix := "getEnv"
		if strings.HasPrefix(expr, "getenv") {
			prefix = "getenv"
		}
		arg := strings.TrimSpace(strings.TrimPrefix(expr, prefix))
		arg = strings.Trim(arg, " \"'")
		return os.Getenv(arg), nil
	}

	if val, ok := vars[expr]; ok {
		return val, nil
	}

	parts := strings.SplitN(expr, ".", 2)
	if len(parts) == 2 {
		module := parts[0]
		accessor := parts[1]

		key := module + "." + accessor
		if val, ok := vars[key]; ok {
			return val, nil
		}

		if modVars, ok := vars[module].(map[string]interface{}); ok {
			if val, ok := modVars[accessor]; ok {
				return val, nil
			}
		}
	}

	return "{{" + expr + "}}", nil
}

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2 (RFC 4122)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	), nil
}

func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random hex: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hurlFormatToGo converts Java-style date format patterns (as used in Hurl/README)
// to Go time layout strings. Also supports strftime-style % patterns.
func hurlFormatToGo(format string) string {
	// Java-style replacements (as shown in README examples)
	javaReplacements := []struct {
		java string
		goFmt string
	}{
		{"yyyy", "2006"},
		{"yy", "06"},
		{"MMMM", "January"},
		{"MMM", "Jan"},
		{"MM", "01"},
		{"dd", "02"},
		{"HH", "15"},
		{"mm", "04"},
		{"ss", "05"},
		{"SSS", "000"},
		{"ZZZZ", "-07:00"},
		{"ZZZ", "-0700"},
		{"Z", "Z07:00"},
	}

	result := format
	for _, r := range javaReplacements {
		result = strings.ReplaceAll(result, r.java, r.goFmt)
	}

	// Also support strftime-style for compatibility
	strftimeReplacements := []struct {
		strftime string
		goFmt   string
	}{
		{"%Y", "2006"},
		{"%m", "01"},
		{"%d", "02"},
		{"%H", "15"},
		{"%M", "04"},
		{"%S", "05"},
		{"%z", "-0700"},
		{"%Z", "MST"},
	}
	for _, r := range strftimeReplacements {
		result = strings.ReplaceAll(result, r.strftime, r.goFmt)
	}

	return result
}
