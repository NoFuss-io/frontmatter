// Package store defines the pluggable data-source backend for fm.
// Third-party store authors import only this package — never internal/.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store is the pluggable data-source backend. The executor calls it to
// enumerate items, read their fields, write mutations back, and label them
// in output tables.
type Store interface {
	// Enumerate resolves FROM-clause pattern tokens into opaque item keys.
	// File stores receive glob strings; API stores receive domain-specific
	// identifiers (project keys, JQL, URLs, …). The store interprets them.
	Enumerate(patterns []string, opts EnumOptions) ([]string, error)

	// Read returns the field map for one item. The key is a value previously
	// returned by Enumerate.
	Read(key string) (map[string]any, error)

	// Write persists a mutated field map. Only called when a mutation
	// statement succeeded and DryRun is false.
	Write(key string, fields map[string]any) error

	// Label returns a human-readable display name for the key, used as the
	// row identifier in output tables (the "filename" column).
	Label(key string) string
}

// EnumOptions carries per-enumeration settings that the store may consult.
type EnumOptions struct {
	// IncludeHidden requests that items whose names begin with '.' not be
	// filtered out. File stores apply this to basenames; API stores ignore it.
	IncludeHidden bool
}

// Format handles reading and writing a single file. Glob expansion and hidden-
// file filtering are provided by FileStore, so Format authors only need to
// care about the file contents.
type Format interface {
	Read(path string) (map[string]any, error)
	Write(path string, fields map[string]any) error
}

// FileStore is a concrete Store for any file-based format. It implements
// Enumerate (glob expansion + hidden-file filter) once, and delegates
// Read/Write to the supplied Format. Label returns the path unchanged.
type FileStore struct {
	Fmt Format
}

func (fs FileStore) Enumerate(patterns []string, opts EnumOptions) ([]string, error) {
	paths, err := expandGlobs(patterns)
	if err != nil {
		return nil, err
	}
	if !opts.IncludeHidden {
		paths = filterHidden(paths, patterns)
	}
	return paths, nil
}

func (fs FileStore) Read(key string) (map[string]any, error) {
	return fs.Fmt.Read(key)
}

func (fs FileStore) Write(key string, fields map[string]any) error {
	return fs.Fmt.Write(key, fields)
}

func (fs FileStore) Label(key string) string {
	return key
}

// expandGlobs expands pattern tokens to file paths. Bare paths that exist are
// passed through; tokens containing glob metacharacters are expanded via
// filepath.Glob. A bare-path token that does not exist is an error.
func expandGlobs(patterns []string) ([]string, error) {
	var paths []string
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?[") {
			matched, err := filepath.Glob(p)
			if err != nil {
				return nil, fmt.Errorf("glob %q: %w", p, err)
			}
			for _, m := range matched {
				fi, err := os.Stat(m)
				if err == nil && fi.Mode().IsRegular() {
					paths = append(paths, m)
				}
			}
			continue
		}
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("no such file or pattern: %s", p)
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// filterHidden drops paths whose basename begins with '.', except those that
// match a bare-path (non-glob) token, which the user named explicitly.
func filterHidden(paths []string, patterns []string) []string {
	bare := make(map[string]bool, len(patterns))
	for _, g := range patterns {
		if !strings.ContainsAny(g, "*?[") {
			bare[g] = true
		}
	}
	out := paths[:0]
	for _, p := range paths {
		if bare[p] {
			out = append(out, p)
			continue
		}
		base := filepath.Base(p)
		if strings.HasPrefix(base, ".") {
			continue
		}
		out = append(out, p)
	}
	return out
}
