package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wei-lli/hurlx/importer"
	"github.com/wei-lli/hurlx/runner"
	"github.com/wei-lli/hurlx/tmpl"
)

var (
	version = "1.0.13"

	flagVariable        arrayFlags
	flagVariablesFile   string
	flagSecret          arrayFlags
	flagInsecure        bool
	flagLocation        bool
	flagMaxRedirs       int
	flagVerbose         bool
	flagVeryVerbose     bool
	flagInclude         bool
	flagCompressed      bool
	flagIgnoreAsserts   bool
	flagContinueOnError bool
	flagTest            bool
	flagFromEntry       int
	flagToEntry         int
	flagOutput          string
	flagFileRoot        string
	flagProxy           string
	flagHTTPVersion     string
	flagUser            string
	flagUserAgent       string
	flagTimeout         string
	flagConnectTimeout  string
	flagDelay           string
	flagRetry           int
	flagRetryInterval   string
	flagIPv4            bool
	flagIPv6            bool
	flagJSON            bool
	flagColor           bool
	flagNoColor         bool
	flagNoOutput        bool
	flagPretty          bool
	flagNoPretty        bool
	flagRepeat          int
	flagTrace           bool
)

type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ", ")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func init() {
	flag.Var(&flagVariable, "variable", "Define variable (name=value)")
	flag.Var(&flagVariable, "V", "Define variable (name=value)")
	flag.StringVar(&flagVariablesFile, "variables-file", "", "Define variables file")
	flag.Var(&flagSecret, "secret", "Define secret (name=value)")
	flag.BoolVar(&flagInsecure, "insecure", false, "Allow insecure SSL connections")
	flag.BoolVar(&flagInsecure, "k", false, "Allow insecure SSL connections")
	flag.BoolVar(&flagLocation, "location", false, "Follow redirects")
	flag.BoolVar(&flagLocation, "L", false, "Follow redirects")
	flag.IntVar(&flagMaxRedirs, "max-redirs", 50, "Maximum number of redirects")
	flag.BoolVar(&flagVerbose, "verbose", false, "Verbose output")
	flag.BoolVar(&flagVerbose, "v", false, "Verbose output")
	flag.BoolVar(&flagVeryVerbose, "very-verbose", false, "More verbose output")
	flag.BoolVar(&flagInclude, "include", false, "Include HTTP headers in output")
	flag.BoolVar(&flagInclude, "i", false, "Include HTTP headers in output")
	flag.BoolVar(&flagCompressed, "compressed", false, "Request compressed response")
	flag.BoolVar(&flagIgnoreAsserts, "ignore-asserts", false, "Ignore all asserts")
	flag.BoolVar(&flagContinueOnError, "continue-on-error", false, "Continue on assert errors")
	flag.BoolVar(&flagTest, "test", false, "Test mode")
	flag.IntVar(&flagFromEntry, "from-entry", 0, "Start from entry number")
	flag.IntVar(&flagToEntry, "to-entry", 0, "End at entry number")
	flag.StringVar(&flagOutput, "output", "", "Output file")
	flag.StringVar(&flagOutput, "o", "", "Output file")
	flag.StringVar(&flagFileRoot, "file-root", "", "Root directory for file references")
	flag.StringVar(&flagProxy, "proxy", "", "Proxy server")
	flag.StringVar(&flagProxy, "x", "", "Proxy server")
	flag.StringVar(&flagHTTPVersion, "http-version", "", "HTTP version (1.0, 1.1, 2, 3)")
	flag.StringVar(&flagUser, "user", "", "Basic authentication (user:password)")
	flag.StringVar(&flagUser, "u", "", "Basic authentication (user:password)")
	flag.StringVar(&flagUserAgent, "user-agent", "hurlx/"+version, "User-Agent string")
	flag.StringVar(&flagTimeout, "max-time", "", "Maximum time per request")
	flag.StringVar(&flagTimeout, "m", "", "Maximum time per request")
	flag.StringVar(&flagConnectTimeout, "connect-timeout", "", "Connection timeout")
	flag.StringVar(&flagDelay, "delay", "", "Delay between requests")
	flag.IntVar(&flagRetry, "retry", 0, "Maximum number of retries")
	flag.StringVar(&flagRetryInterval, "retry-interval", "1000ms", "Retry interval")
	flag.BoolVar(&flagIPv4, "ipv4", false, "Use IPv4 only")
	flag.BoolVar(&flagIPv4, "4", false, "Use IPv4 only")
	flag.BoolVar(&flagIPv6, "ipv6", false, "Use IPv6 only")
	flag.BoolVar(&flagIPv6, "6", false, "Use IPv6 only")
	flag.BoolVar(&flagJSON, "json", false, "JSON output")
	flag.BoolVar(&flagColor, "color", false, "Colorize output")
	flag.BoolVar(&flagNoColor, "no-color", false, "No color output")
	flag.BoolVar(&flagNoOutput, "no-output", false, "Suppress output")
	flag.BoolVar(&flagPretty, "pretty", false, "Prettify output")
	flag.BoolVar(&flagNoPretty, "no-pretty", false, "No prettify")
	flag.IntVar(&flagRepeat, "repeat", 1, "Repeat count")
	flag.BoolVar(&flagTrace, "trace", false, "Trace each chain step result as JSON")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "hurlx %s - Run and test HTTP requests with import/export support\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: hurlx [options] [FILE...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  hurlx example.hurlx\n")
		fmt.Fprintf(os.Stderr, "  hurlx --test *.hurlx\n")
		fmt.Fprintf(os.Stderr, "  hurlx --variable host=example.org api.hurlx\n")
	}

	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			flag.Usage()
			os.Exit(1)
		}
		files = []string{"/dev/stdin"}
	}

	variables := tmpl.NewVariables()

	for _, v := range flagVariable {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			variables.Set(parts[0], parts[1])
		}
	}

	if flagVariablesFile != "" {
		data, err := os.ReadFile(flagVariablesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading variables file: %v\n", err)
			os.Exit(2)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				variables.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	for _, v := range flagSecret {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			variables.Set(parts[0], parts[1])
		}
	}

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "HURLX_VARIABLE_") {
			key := strings.TrimPrefix(env, "HURLX_VARIABLE_")
			if idx := strings.Index(key, "="); idx >= 0 {
				variables.Set(key[:idx], key[idx+1:])
			}
		}
	}

	var timeout time.Duration
	if flagTimeout != "" {
		timeout = runner.ParseDuration(flagTimeout)
	}

	var connectTimeout time.Duration
	if flagConnectTimeout != "" {
		connectTimeout = runner.ParseDuration(flagConnectTimeout)
	}

	exitCode := 0

	for repeat := 0; repeat < flagRepeat; repeat++ {
		for _, file := range expandFiles(files) {
			resolver := importer.NewResolver(flagFileRoot)
			resolved, err := resolver.Resolve(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				exitCode = 2
				continue
			}

			for name, value := range resolved.AllExports {
				if _, ok := variables.Get(name); !ok {
					variables.Set(name, value)
				}
			}

			opts := runner.RunOptions{
				Variables:       variables.Clone(),
				Insecure:        flagInsecure,
				FollowRedirect:  flagLocation,
				MaxRedirects:    flagMaxRedirs,
				Timeout:         timeout,
				ConnectTimeout:  connectTimeout,
				Compressed:      flagCompressed,
				Verbose:         flagVerbose,
				VeryVerbose:     flagVeryVerbose,
				Include:         flagInclude,
				IgnoreAsserts:   flagIgnoreAsserts,
				ContinueOnError: flagContinueOnError,
				FromEntry:       flagFromEntry,
				ToEntry:         flagToEntry,
				Output:          flagOutput,
				FileRoot:        flagFileRoot,
				Proxy:           flagProxy,
				HTTPVersion:     flagHTTPVersion,
				User:            flagUser,
				UserAgent:       flagUserAgent,
				Trace:           flagTrace,
			}

			r := runner.NewRunner(opts)
			result, err := r.Run(resolved.AllEntries)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error in %s: %v\n", file, err)
				if !flagContinueOnError {
					exitCode = 4
					continue
				}
			}

			if !result.Success {
				exitCode = 4
			}

			if flagTest {
				printTestResult(file, result)
			} else if flagJSON {
				printJSONResult(file, result)
			} else {
				printNormalResult(result)
			}
		}
	}

	os.Exit(exitCode)
}

func expandFiles(files []string) []string {
	var expanded []string
	for _, f := range files {
		if strings.Contains(f, "*") || strings.Contains(f, "?") {
			matches, err := filepath.Glob(f)
			if err == nil && len(matches) > 0 {
				expanded = append(expanded, matches...)
			} else {
				expanded = append(expanded, f)
			}
		} else {
			info, err := os.Stat(f)
			if err == nil && info.IsDir() {
				filepath.WalkDir(f, func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() && (strings.HasSuffix(path, ".hurlx") || strings.HasSuffix(path, ".hurl")) {
						expanded = append(expanded, path)
					}
					return nil
				})
			} else {
				expanded = append(expanded, f)
			}
		}
	}
	return expanded
}

func printNormalResult(result *runner.RunResult) {
	if flagNoOutput {
		return
	}
	if len(result.Entries) == 0 {
		return
	}

	last := result.Entries[len(result.Entries)-1]
	if last.Response == nil {
		return
	}

	if flagOutput != "" && flagOutput != "-" {
		body := last.Body
		if shouldUsePretty(last.Response.Header.Get("Content-Type")) && isValidJSON(body) {
			if prettified, err := prettifyJSON(body); err == nil {
				body = prettified
			}
		}
		if err := os.WriteFile(flagOutput, body, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		}
	} else {
		if flagInclude && last.Response != nil {
			fmt.Printf("%s %d\n", last.Response.Proto, last.Response.StatusCode)
			for k, v := range last.Response.Header {
				fmt.Printf("%s: %s\n", k, strings.Join(v, ", "))
			}
			fmt.Println()
		}
		body := formatBody(last.Body, last.Response.Header.Get("Content-Type"))
		os.Stdout.Write(body)
		if len(body) > 0 && body[len(body)-1] != '\n' {
			fmt.Println()
		}
	}
}

func printTestResult(file string, result *runner.RunResult) {
	status := "SUCCESS"
	if !result.Success {
		status = "FAILURE"
	}

	fmt.Printf("%s: %s", filepath.Base(file), status)
	if !result.Success {
		for _, e := range result.Entries {
			if e.Error != nil {
				fmt.Printf(" (entry %d: %s)", e.EntryIndex+1, e.Error)
			}
		}
	}
	fmt.Println()
}

func printJSONResult(file string, result *runner.RunResult) {
	type jsonEntry struct {
		Entry    int    `json:"entry"`
		Method   string `json:"method,omitempty"`
		URL      string `json:"url,omitempty"`
		Status   int    `json:"status,omitempty"`
		Duration int64  `json:"duration_ms,omitempty"`
		Error    string `json:"error,omitempty"`
		Body     string `json:"body,omitempty"`
	}

	type jsonOutput struct {
		File    string      `json:"file"`
		Success bool        `json:"success"`
		Entries []jsonEntry `json:"entries"`
	}

	output := jsonOutput{
		File:    file,
		Success: result.Success,
	}

	for _, e := range result.Entries {
		entry := jsonEntry{
			Entry:    e.EntryIndex + 1,
			Duration: int64(e.Duration / time.Millisecond),
		}
		if e.Request != nil {
			entry.Method = e.Request.Method
			entry.URL = e.Request.URL.String()
		}
		if e.Response != nil {
			entry.Status = e.Response.StatusCode
		}
		if e.Error != nil {
			entry.Error = e.Error.Error()
		}
		if e.Body != nil {
			entry.Body = string(e.Body)
		}
		output.Entries = append(output.Entries, entry)
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON output: %v\n", err)
		return
	}
	if shouldUseColor() {
		fmt.Println(string(colorizeJSON(data)))
	} else {
		fmt.Println(string(data))
	}
}
