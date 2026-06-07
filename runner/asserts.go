package runner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wei-lli/hurlx/ast"
	"github.com/wei-lli/hurlx/tmpl"
)

func (r *Runner) checkAssert(index int, assert ast.Assert, value interface{}, exists bool, vars tmpl.Variables) error {
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
			rendered, err := tmpl.Render(assertVal.Str, vars)
			if err == nil && rendered != assertVal.Str {
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
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
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
	needle := formatAssertValue(expected)
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
		if expected.Type == ast.ValueString {
			if fmt.Sprintf("%v", item) == expected.Str {
				found = true
				break
			}
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
	prefix := formatAssertValue(expected)
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
	suffix := formatAssertValue(expected)
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
	pattern := formatAssertValue(expected)
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

func regexpMatch(pattern string, s string) (bool, error) {
	if len(pattern) > MaxRegexPatternLen {
		return false, fmt.Errorf("regex: pattern exceeds maximum length of %d", MaxRegexPatternLen)
	}
	re, err := compileRegex(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(s), nil
}
