package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"unicode"
)

func isJSON(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}

func isValidJSON(data []byte) bool {
	if !isJSON(data) {
		return false
	}
	var v interface{}
	return json.Unmarshal(data, &v) == nil
}

func prettifyJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return data, err
	}
	return json.MarshalIndent(v, "", "  ")
}

func colorizeJSON(data []byte) []byte {
	var buf bytes.Buffer
	i := 0
	n := len(data)

	for i < n {
		ch := data[i]

		switch {
		case ch == '"':
			j := i + 1
			for j < n {
				if data[j] == '\\' && j+1 < n {
					j += 2
					continue
				}
				if data[j] == '"' {
					j++
					break
				}
				j++
			}
			token := data[i:j]

			if isKey(data, i) {
				buf.WriteString("\x1b[36m")
				buf.Write(token)
				buf.WriteString("\x1b[0m")
			} else {
				buf.WriteString("\x1b[32m")
				buf.Write(token)
				buf.WriteString("\x1b[0m")
			}
			i = j

		case ch == '-' || (ch >= '0' && ch <= '9'):
			j := i
			if ch == '-' {
				j++
			}
			for j < n && (data[j] >= '0' && data[j] <= '9' || data[j] == '.' || data[j] == 'e' || data[j] == 'E' || data[j] == '+' || data[j] == '-') {
				j++
			}
			buf.WriteString("\x1b[33m")
			buf.Write(data[i:j])
			buf.WriteString("\x1b[0m")
			i = j

		case i+3 < n && string(data[i:i+4]) == "true":
			buf.WriteString("\x1b[35m")
			buf.WriteString("true")
			buf.WriteString("\x1b[0m")
			i += 4

		case i+4 < n && string(data[i:i+5]) == "false":
			buf.WriteString("\x1b[35m")
			buf.WriteString("false")
			buf.WriteString("\x1b[0m")
			i += 5

		case i+3 < n && string(data[i:i+4]) == "null":
			buf.WriteString("\x1b[90m")
			buf.WriteString("null")
			buf.WriteString("\x1b[0m")
			i += 4

		default:
			buf.WriteByte(ch)
			i++
		}
	}

	return buf.Bytes()
}

func isKey(data []byte, quotePos int) bool {
	j := quotePos + 1
	for j < len(data) {
		if data[j] == '\\' && j+1 < len(data) {
			j += 2
			continue
		}
		if data[j] == '"' {
			break
		}
		j++
	}
	k := j + 1
	for k < len(data) && unicode.IsSpace(rune(data[k])) {
		k++
	}
	return k < len(data) && data[k] == ':'
}

func shouldUseColor() bool {
	if flagColor {
		return true
	}
	if flagNoColor {
		return false
	}
	return isTerminal(os.Stdout)
}

func shouldUsePretty(contentType string) bool {
	if flagPretty {
		return true
	}
	if flagNoPretty {
		return false
	}
	return strings.Contains(strings.ToLower(contentType), "json")
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func formatBody(body []byte, contentType string) []byte {
	if !isValidJSON(body) {
		return body
	}

	result := body

	if shouldUsePretty(contentType) {
		prettified, err := prettifyJSON(body)
		if err == nil {
			result = prettified
		}
	}

	if shouldUseColor() {
		result = colorizeJSON(result)
	}

	return result
}
