// Package markdown implements store.Format for Markdown files with YAML
// frontmatter, and for source files that carry the same block in a comment.
package markdown

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/nofuss-io/frontmatter/store"
)

// New returns a store.Store backed by Markdown files with YAML frontmatter.
// A fresh Format (and its body sidecar) is created on each call, so the
// sidecar lifetime is tied to the single Run invocation.
func New() store.Store {
	return store.FileStore{Fmt: &Format{}, DefaultExts: Extensions()}
}

// Format implements store.Format for files with YAML frontmatter: Markdown
// files, and source files with the block inside a comment (see languages).
// The docs sync.Map caches the parsed Document keyed by path so that Write can
// reconstruct the full file, in its original comment style and line endings,
// without data loss.
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
	docs sync.Map // key: string path → value: Document
}

// Read parses the file at path, stashes the Document in the sidecar for a
// later Write call, and returns its frontmatter field map. A file without a
// block yields an empty map.
func (f *Format) Read(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(raw, 0) >= 0 || !utf8.Valid(raw) {
		return nil, errors.New("not a text file")
	}
	doc, err := Parse(path, raw)
	if err != nil {
		return nil, err
	}
	f.docs.Store(path, doc)
	return doc.FrontMatter, nil
}

// Write renders fields into the Document stashed by the preceding Read and
// writes the result to disk, replacing the original file. Only the
// frontmatter block changes; everything around it is written back verbatim.
func (f *Format) Write(path string, fields map[string]any) error {
	doc := Document{Body: string(LF), LineEndings: LF}
	if v, ok := f.docs.Load(path); ok {
		doc = v.(Document)
	}
	doc.FrontMatter = fields
	out, err := Render(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

func marshalFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		out[k] = marshalValue(v)
	}
	return out
}

// marshalValue converts Go values to yaml.Node where needed so the encoder
// writes them in their original YAML form:
//   - time.Time uses !!timestamp so dates stay as 2006-01-02, not RFC3339.
//   - string uses !!str so yaml.v3's isOldBool guard (which quotes "y", "n",
//     "on", "off", etc. for YAML 1.1 compatibility) is bypassed; the encoder
//     still quotes strings that resolve to non-string types (true, null, 1…).
func marshalValue(v any) any {
	switch val := v.(type) {
	case time.Time:
		s := val.Format(time.RFC3339)
		if val.Hour() == 0 && val.Minute() == 0 && val.Second() == 0 && val.Nanosecond() == 0 {
			s = val.Format("2006-01-02")
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: s}
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val}
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = marshalValue(e)
		}
		return out
	}
	return v
}
