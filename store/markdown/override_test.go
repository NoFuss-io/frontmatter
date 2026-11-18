package markdown_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nofuss-io/frontmatter/store"
	"github.com/nofuss-io/frontmatter/store/markdown"
)

func TestNewWithOverride(t *testing.T) {
	tests := []struct {
		name     string
		override markdown.Override
		file     string
		input    string
		want     map[string]any
	}{
		{
			name:     "lang by name ignores extension",
			override: markdown.Override{Lang: "python"},
			file:     "a.md",
			input:    "# ---\n# a: 1\n# ---\n",
			want:     map[string]any{"a": 1},
		},
		{
			name:     "lang by extension",
			override: markdown.Override{Lang: ".js"},
			file:     "deploy",
			input:    "// ---\n// a: 1\n// ---\n",
			want:     map[string]any{"a": 1},
		},
		{
			name:     "lang markdown ignores comments",
			override: markdown.Override{Lang: "markdown"},
			file:     "a.py",
			input:    "# ---\n# a: 1\n# ---\n",
			want:     map[string]any{},
		},
		{
			name:     "custom line comment",
			override: markdown.Override{LineComment: ";;"},
			file:     "a.cfg",
			input:    ";; ---\n;; a: 1\n;; ---\n",
			want:     map[string]any{"a": 1},
		},
		{
			name:     "custom block comment",
			override: markdown.Override{BlockOpen: "{{/*", BlockClose: "*/}}"},
			file:     "a.tmpl",
			input:    "{{/*---\na: 1\n---*/}}\n",
			want:     map[string]any{"a": 1},
		},
		{
			name:     "block close defaults to block open",
			override: markdown.Override{BlockOpen: "%%"},
			file:     "a.txt",
			input:    "%%\n---\na: 1\n---\n%%\n",
			want:     map[string]any{"a": 1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, t.TempDir(), tc.file, tc.input)
			s, err := markdown.NewWithOverride(tc.override)
			if err != nil {
				t.Fatalf("NewWithOverride: %v", err)
			}
			got, err := s.Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNewWithOverride_Errors(t *testing.T) {
	for _, o := range []markdown.Override{
		{Lang: "klingon"},
		{Lang: "python", LineComment: "#"},
		{Lang: "python", BlockOpen: "/*"},
		{BlockClose: "*/"},
	} {
		if _, err := markdown.NewWithOverride(o); err == nil {
			t.Errorf("%+v: expected error", o)
		}
	}
}

func TestNewWithOverride_GlobMatchesAllFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.md", "deploy", "b.cfg"} {
		writeFile(t, dir, name, "")
	}
	glob := filepath.Join(dir, "*")
	count := func(s store.Store) int {
		paths, err := s.Enumerate([]string{glob}, store.EnumOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return len(paths)
	}
	if n := count(markdown.New()); n != 1 {
		t.Errorf("default store: got %d files, want 1", n)
	}
	s, err := markdown.NewWithOverride(markdown.Override{Lang: "shell"})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(s); n != 3 {
		t.Errorf("override store: got %d files, want 3", n)
	}
}
