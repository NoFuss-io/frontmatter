// Package markdown implements store.Format for Markdown files with YAML
// frontmatter, and for source files that carry the same block in a comment.
package markdown

import (
	"bytes"
	"errors"
	"os"
	"strings"
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
	f := &Format{}
	return fileStore{store.FileStore{Fmt: f, DefaultExts: Extensions()}, f}
}

// fileStore is store.FileStore plus a record of which paths the user named
// explicitly (rather than matched by a glob), which Format.Write consults.
type fileStore struct {
	store.FileStore
	f *Format
}

func (s fileStore) Enumerate(patterns []string, opts store.EnumOptions) ([]string, error) {
	for _, p := range patterns {
		if !strings.ContainsAny(p, "*?[") {
			s.f.explicit.Store(p, true)
		}
	}
	return s.FileStore.Enumerate(patterns, opts)
}

// ErrNoBlock is returned by Write for a source file without a frontmatter
// block that was only matched by a glob. Inserting a block there would edit
// files the user may not have meant to touch (scripts, stylesheets, …).
var ErrNoBlock = errors.New("no frontmatter block; name the file explicitly or pass --lang to insert one")

// parsed is the sidecar entry for one file.
type parsed struct {
	doc      Document
	found    bool // the file had a block
	markdown bool // parsed with Markdown rules
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
	docs sync.Map // key: string path → value: parsed
	// explicit holds paths named without glob characters (see fileStore).
	explicit sync.Map // key: string path → value: true
	// override, when set, is used for every file instead of languageFor.
	override *language
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
	lang := languageFor(path, string(raw))
	if f.override != nil {
		lang = *f.override
	}
	doc, found, err := parseAs(path, raw, lang)
	if err != nil {
		return nil, err
	}
	f.docs.Store(path, parsed{doc: doc, found: found, markdown: lang.markdown})
	return doc.FrontMatter, nil
}

// Write renders fields into the Document stashed by the preceding Read and
// writes the result to disk, replacing the original file. Only the
// frontmatter block changes; everything around it is written back verbatim.
//
// A new block is inserted into a Markdown file freely, but into a source file
// only if the user named it explicitly or set an override; otherwise Write
// returns ErrNoBlock and leaves the file untouched.
func (f *Format) Write(path string, fields map[string]any) error {
	doc := Document{Body: string(LF), LineEndings: LF}
	if v, ok := f.docs.Load(path); ok {
		p := v.(parsed)
		if !p.found && !p.markdown && f.override == nil {
			if _, named := f.explicit.Load(path); !named {
				return ErrNoBlock
			}
		}
		doc = p.doc
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
