package lib

import (
	"fmt"
	"io"
	"os"
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
