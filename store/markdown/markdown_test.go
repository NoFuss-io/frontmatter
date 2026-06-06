package markdown_test

import (
	"os"
	"path/filepath"
	"testing"

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
