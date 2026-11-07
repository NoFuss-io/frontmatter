package markdown_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nofuss-io/frontmatter/store/markdown"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFormat_Read_ValidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "note.md", "---\ntitle: Hello\ntags:\n  - a\n  - b\n---\nBody text.\n")

	s := markdown.New()
	fields, err := s.Read(path)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if fields["title"] != "Hello" {
		t.Errorf("expected title=Hello, got %v", fields["title"])
	}
	tags, ok := fields["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Errorf("expected tags slice of length 2, got %v", fields["tags"])
	}
}

func TestFormat_Read_CRLFLineEndings(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "crlf.md", "---\r\ntitle: Hello\r\ntags:\r\n  - a\r\n  - b\r\n---\r\nBody.\r\n")

	s := markdown.New()
	fields, err := s.Read(path)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if fields["title"] != "Hello" {
		t.Errorf("expected title=Hello, got %v", fields["title"])
	}
	tags, ok := fields["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Errorf("expected tags slice of length 2, got %v", fields["tags"])
	}
}

func TestFormat_Read_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "plain.md", "Just a body.\n")

	s := markdown.New()
	fields, err := s.Read(path)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty map for file without frontmatter, got %v", fields)
	}
}

func TestFormat_ReadWriteRead_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := "---\ntitle: Original\ncount: 1\n---\nThe body text.\n"
	path := writeFile(t, dir, "round.md", original)

	s := markdown.New()

	// Read
	fields, err := s.Read(path)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Mutate
	fields["title"] = "Updated"
	fields["count"] = 2

	// Write
	if err := s.Write(path, fields); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Read again with a fresh store to avoid sidecar interference
	s2 := markdown.New()
	fields2, err := s2.Read(path)
	if err != nil {
		t.Fatalf("second Read error: %v", err)
	}

	if fields2["title"] != "Updated" {
		t.Errorf("expected title=Updated after round-trip, got %v", fields2["title"])
	}

	// Verify body is preserved on disk
	raw, _ := os.ReadFile(path)
	content := string(raw)
	if !containsStr(content, "The body text.") {
		t.Errorf("body not preserved after Write; file content:\n%s", content)
	}
}

func TestWrite_FieldTypes(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "date preserved as date",
			fields: map[string]any{"birthday": time.Date(2001, 2, 3, 0, 0, 0, 0, time.UTC)},
			want:   "---\nbirthday: 2001-02-03\n---\n",
		},
		{
			name:   "plain string unquoted",
			fields: map[string]any{"a": "x", "b": "\"y\"", "c": "z \"z\" z"},
			want:   "---\na: x\nb: '\"y\"'\nc: z \"z\" z\n---\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "note.md", "---\n_: _\n---\n")
			s := markdown.New()
			if _, err := s.Read(path); err != nil {
				t.Fatalf("Read: %v", err)
			}
			if err := s.Write(path, tc.fields); err != nil {
				t.Fatalf("Write: %v", err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			if got := string(raw); got != tc.want {
				t.Errorf("file content:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findStr(s, sub))
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// fixtureCases cover YAML frontmatter in Markdown and in source-file comments
// (the examples from issue #21), in both LF and CRLF variants. Each case
// lives in testdata/<dir>/ as input.<ext> and want.<ext>: want is the file
// after setting author to "john.roe". The encoder sorts keys, so created_at
// moves before description in the rewritten block.
//
// Head and Body hold raw bytes, so CRLF cases carry "\r\n" in them.
//
// https://github.com/NoFuss-io/frontmatter/issues/21
var fixtureCases = []struct {
	dir  string
	ext  string
	want markdown.Document
}{
	{
		dir: "markdown", ext: "md",
		want: markdown.Document{
			FrontMatter: fixtureFields("a markdown note"),
			Body:        "\nBody text.\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "markdown-crlf", ext: "md",
		want: markdown.Document{
			FrontMatter: fixtureFields("a markdown note"),
			Body:        "\r\nBody text.\r\n",
			LineEndings: markdown.CRLF,
		},
	},
	{
		dir: "python-hash", ext: "py",
		want: markdown.Document{
			FrontMatter: fixtureFields("exports sessions to Excel"),
			LinePrefix:  "# ",
			Body:        "\nprint(\"hi\")\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "python-hash-crlf", ext: "py",
		want: markdown.Document{
			FrontMatter: fixtureFields("exports sessions to Excel"),
			LinePrefix:  "# ",
			Body:        "\r\nprint(\"hi\")\r\n",
			LineEndings: markdown.CRLF,
		},
	},
	{
		dir: "python-docstring", ext: "py",
		want: markdown.Document{
			FrontMatter: fixtureFields("exports sessions to Excel"),
			Head:        `"""`,
			Body:        "\"\"\"\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "bash-shebang", ext: "sh",
		want: markdown.Document{
			FrontMatter: fixtureFields("rotates the log files"),
			Head:        "#!/usr/bin/env bash\n",
			LinePrefix:  "# ",
			Body:        "\nset -euo pipefail\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "ruby-magic-comment", ext: "rb",
		want: markdown.Document{
			FrontMatter: fixtureFields("imports users from CSV"),
			Head:        "#!/usr/bin/env ruby\n# frozen_string_literal: true\n",
			LinePrefix:  "# ",
			Body:        "\nrequire \"csv\"\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "ruby-begin-end", ext: "rb",
		want: markdown.Document{
			FrontMatter: fixtureFields("imports users from CSV"),
			Head:        "=begin\n",
			Body:        "\n=end\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "javascript-slash", ext: "js",
		want: markdown.Document{
			FrontMatter: fixtureFields("syncs the cache with the API"),
			Head:        "#!/usr/bin/env node\n",
			LinePrefix:  "// ",
			Body:        "\nimport fs from \"node:fs\";\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "javascript-block", ext: "js",
		want: markdown.Document{
			FrontMatter: fixtureFields("syncs the cache with the API"),
			Head:        "/*",
			Body:        "*/\nimport fs from \"node:fs\";\n",
			LineEndings: markdown.LF,
		},
	},
	{
		dir: "javascript-block-crlf", ext: "js",
		want: markdown.Document{
			FrontMatter: fixtureFields("syncs the cache with the API"),
			Head:        "/*",
			Body:        "*/\r\nimport fs from \"node:fs\";\r\n",
			LineEndings: markdown.CRLF,
		},
	},
}

// fixtureFields is the frontmatter every input fixture carries; only the
// description differs between them.
func fixtureFields(description string) map[string]any {
	return map[string]any{
		"author":      "jane.doe",
		"description": description,
		"created_at":  time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
	}
}

// copyFixture copies testdata/<dir>/input.<ext> into a temp dir and returns
// its path, so tests that write never touch the fixture itself.
func copyFixture(t *testing.T, dir, ext string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", dir, "input."+ext))
	if err != nil {
		t.Fatal(err)
	}
	return writeFile(t, t.TempDir(), "input."+ext, string(raw))
}

func TestParse_Fixtures(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.dir, func(t *testing.T) {
			path := filepath.Join("testdata", tc.dir, "input."+tc.ext)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := markdown.Parse(path, raw)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Document:\ngot:  %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestFormat_Read_Fixtures(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.dir, func(t *testing.T) {
			fields, err := markdown.New().Read(copyFixture(t, tc.dir, tc.ext))
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if !reflect.DeepEqual(fields, tc.want.FrontMatter) {
				t.Errorf("fields:\ngot:  %#v\nwant: %#v", fields, tc.want.FrontMatter)
			}
		})
	}
}

func TestFormat_Write_Fixtures(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.dir, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", tc.dir, "want."+tc.ext))
			if err != nil {
				t.Fatal(err)
			}
			path := copyFixture(t, tc.dir, tc.ext)
			s := markdown.New()
			fields, err := s.Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			fields["author"] = "john.roe"
			if err := s.Write(path, fields); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("file content:\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}
