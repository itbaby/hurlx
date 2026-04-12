package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wei-lli/hurlx/ast"
	"github.com/wei-lli/hurlx/parser"
)

type ResolvedFile struct {
	File       *ast.File
	FilePath   string
	Imports    map[string]*ResolvedFile
	AllEntries []ast.Entry
	AllExports map[string]string
}

type Resolver struct {
	cache    map[string]*ResolvedFile
	fileRoot string
}

func NewResolver(fileRoot string) *Resolver {
	return &Resolver{
		cache:    make(map[string]*ResolvedFile),
		fileRoot: fileRoot,
	}
}

func (r *Resolver) Resolve(path string) (*ResolvedFile, error) {
	if path == "/dev/stdin" || path == "-" {
		return r.resolveStdin()
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	if rf, ok := r.cache[absPath]; ok {
		return rf, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file %s: %w", absPath, err)
	}

	p := parser.NewParser(string(data), absPath)
	file, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", absPath, err)
	}

	return r.buildResolved(file, absPath)
}

func (r *Resolver) resolveStdin() (*ResolvedFile, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	p := parser.NewParser(string(data), "stdin")
	file, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse stdin: %w", err)
	}

	return r.buildResolved(file, "stdin")
}

func (r *Resolver) buildResolved(file *ast.File, filePath string) (*ResolvedFile, error) {
	absPath := filePath
	if rf, ok := r.cache[absPath]; ok {
		return rf, nil
	}

	rf := &ResolvedFile{
		File:       file,
		FilePath:   absPath,
		Imports:    make(map[string]*ResolvedFile),
		AllExports: make(map[string]string),
	}

	for _, exp := range file.Exports {
		rf.AllExports[exp.Name] = exp.Value
	}

	for _, imp := range file.Imports {
		importPath := imp.Path
		if !filepath.IsAbs(importPath) {
			dir := filepath.Dir(absPath)
			if dir == "." || dir == "stdin" {
				dir, _ = os.Getwd()
			}
			importPath = filepath.Join(dir, importPath)
		}

		if !strings.HasSuffix(importPath, ".hurlx") && !strings.HasSuffix(importPath, ".hurl") {
			importPath += ".hurlx"
			if _, err := os.Stat(importPath); os.IsNotExist(err) {
				importPath = strings.TrimSuffix(importPath, ".hurlx") + ".hurl"
			}
		}

		resolved, err := r.Resolve(importPath)
		if err != nil {
			return nil, fmt.Errorf("in file %s: failed to resolve import %s: %w", absPath, imp.Path, err)
		}

		alias := imp.Alias
		if alias == "" {
			alias = strings.TrimSuffix(filepath.Base(imp.Path), filepath.Ext(imp.Path))
		}
		rf.Imports[alias] = resolved

		for name, value := range resolved.AllExports {
			rf.AllExports[name] = value
		}
	}

	var entries []ast.Entry
	seen := make(map[string]bool)
	var addEntries func(imports map[string]*ResolvedFile)
	addEntries = func(imports map[string]*ResolvedFile) {
		for alias, resolved := range imports {
			if seen[alias] {
				continue
			}
			seen[alias] = true
			addEntries(resolved.Imports)
			entries = append(entries, resolved.File.Entries...)
		}
	}
	addEntries(rf.Imports)
	entries = append(entries, file.Entries...)

	rf.AllEntries = entries
	r.cache[absPath] = rf

	return rf, nil
}
