// Package markdown implements store.Format for Markdown files with YAML frontmatter.
package markdown

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/nofuss-io/frontmatter/store"
)

// New returns a store.Store backed by Markdown files with YAML frontmatter.
// A fresh Format (and its body sidecar) is created on each call, so the
// sidecar lifetime is tied to the single Run invocation.
func New() store.Store {
	return store.FileStore{Fmt: &Format{}}
}

// Format implements store.Format for Markdown files with YAML frontmatter.
// The bodies sync.Map caches the non-frontmatter body text keyed by path so
// that Write can reconstruct the full file without data loss.
//
// Body preservation is a general problem for any file-based format: an image
// store must preserve pixel data, an audio store must preserve audio frames,
// etc. The sidecar here is Markdown-specific, but the pattern would be the
// same. A cleaner long-term solution is to move the sidecar up into
// store.FileStore by introducing a File{Fields, Opaque any} type — Format
// returns it from Read and receives it back in Write, FileStore manages the
// map. That would make every Format implementation stateless. It is left as a
// future refactor to keep this first implementation simple.
type Format struct {
	bodies sync.Map // key: string path → value: string body
}

// Read parses the YAML frontmatter from the file at path, stashes the body
// in the sidecar for a later Write call, and returns the frontmatter field map.
// A file without a leading "---\n" block is treated as body-only; an empty map
// is returned and the full file content is stashed as the body.
func (f *Format) Read(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm := make(map[string]any)
	s := string(raw)

	if !strings.HasPrefix(s, "---\n") {
		f.bodies.Store(path, s)
		return fm, nil
	}

	rest := s[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		// Handle trailing "---" with no trailing newline.
		if after, found := strings.CutSuffix(rest, "\n---"); found {
			if err := yaml.Unmarshal([]byte(after), &fm); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			f.bodies.Store(path, "")
			return fm, nil
		}
		// No valid closing fence — treat entire file as body.
		f.bodies.Store(path, s)
		return fm, nil
	}

	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	body := rest[end+5:]
	f.bodies.Store(path, body)
	return fm, nil
}

// Write serializes fields as YAML frontmatter, prepends "---\n…---\n", appends
// the body that was stashed during the preceding Read, and writes the result to
// disk, replacing the original file.
func (f *Format) Write(path string, fields map[string]any) error {
	body := ""
	if v, ok := f.bodies.Load(path); ok {
		body = v.(string)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fields); err != nil {
		return err
	}
	buf.WriteString("---\n")
	buf.WriteString(body)
	return os.WriteFile(path, buf.Bytes(), 0644)
}
