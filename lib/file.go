package lib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FrontMatter map[string]any

type Document struct {
	FrontMatter FrontMatter
	Body        io.Reader
}

type File struct {
	Path        string
	FrontMatter FrontMatter
	Body        string
}

func ReadFile(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &File{Path: path, FrontMatter: make(FrontMatter)}
	s := string(raw)

	if !strings.HasPrefix(s, "---\n") {
		f.Body = s
		return f, nil
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if after, found := strings.CutSuffix(rest, "\n---"); found {
			if err := yaml.Unmarshal([]byte(after), &f.FrontMatter); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return f, nil
		}
		f.Body = s
		return f, nil
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &f.FrontMatter); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	f.Body = rest[end+5:]
	return f, nil
}

func (f *File) Write() error {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f.FrontMatter); err != nil {
		return err
	}
	buf.WriteString("---\n")
	buf.WriteString(f.Body)
	return os.WriteFile(f.Path, buf.Bytes(), 0644)
}

// ExpndGlobs expands the patterns to file paths. Bare paths that exist are
// passed through; tokens containing glob metacharacters are expanded via
// filepath.Glob. A token that resolves to no match is an error.
func ExpandGlobs(globs []string) ([]string, error) {
	var paths []string
	for _, p := range globs {
		if strings.ContainsAny(p, "*?[") {
			matched, err := filepath.Glob(p)
			if err != nil {
				return nil, fmt.Errorf("glob %q: %w", p, err)
			}
			paths = append(paths, matched...)
			continue
		}
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("no such file or pattern: %s", p)
		}
		paths = append(paths, p)
	}
	return paths, nil
}
