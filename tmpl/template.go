package tmpl

import (
	"crypto/rand"
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

func Render(input string, vars Variables) string {
	return templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		result := evaluateExpr(expr, vars)
		return fmt.Sprintf("%v", result)
	})
}

func evaluateExpr(expr string, vars Variables) interface{} {
	expr = strings.TrimSpace(expr)

	switch {
	case expr == "newUuid":
		return generateUUID()
	case expr == "newDate":
		return time.Now().UTC().Format(time.RFC3339Nano)
	case strings.HasPrefix(expr, "getEnv"):
		arg := strings.TrimSpace(strings.TrimPrefix(expr, "getEnv"))
		arg = strings.Trim(arg, " \"'")
		return os.Getenv(arg)
	}

	if val, ok := vars[expr]; ok {
		return val
	}

	parts := strings.SplitN(expr, ".", 2)
	if len(parts) == 2 {
		module := parts[0]
		accessor := parts[1]

		key := module + "." + accessor
		if val, ok := vars[key]; ok {
			return val
		}

		if modVars, ok := vars[module].(map[string]interface{}); ok {
			if val, ok := modVars[accessor]; ok {
				return val
			}
		}
	}

	return "{{" + expr + "}}"
}

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate random bytes: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2 (RFC 4122)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	)
}
