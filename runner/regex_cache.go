package runner

import (
	"regexp"
	"sync"
)

// regexCache caches compiled regex patterns to avoid recompilation
// across multiple entries using the same pattern.
var regexCache struct {
	mu    sync.RWMutex
	cache map[string]*regexp.Regexp
}

func init() {
	regexCache.cache = make(map[string]*regexp.Regexp)
}

// compileRegex returns a cached compiled regex, or compiles and caches a new one.
func compileRegex(pattern string) (*regexp.Regexp, error) {
	regexCache.mu.RLock()
	re, ok := regexCache.cache[pattern]
	regexCache.mu.RUnlock()

	if ok {
		return re, nil
	}

	regexCache.mu.Lock()
	defer regexCache.mu.Unlock()

	// Double-check after acquiring write lock
	if re, ok = regexCache.cache[pattern]; ok {
		return re, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.cache[pattern] = re
	return re, nil
}
