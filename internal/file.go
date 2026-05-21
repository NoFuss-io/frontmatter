package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FilePath = string

type FrontMatter map[string]any

type Document struct {
	FrontMatter FrontMatter
	Body        string
}

// ExpandGlobs expands the patterns to file paths. Bare paths that exist are
// passed through; tokens containing glob metacharacters are expanded via
// filepath.Glob. A bare-path token that does not exist is an error.
func ExpandGlobs(globs []string) ([]FilePath, error) {
	var paths []FilePath
	for _, p := range globs {
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
