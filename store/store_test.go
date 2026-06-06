package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nofuss-io/frontmatter/store"
)

// stubFormat is a no-op Format implementation used for testing FileStore.Enumerate.
type stubFormat struct{}

func (s stubFormat) Read(path string) (map[string]any, error)       { return nil, nil }
func (s stubFormat) Write(path string, fields map[string]any) error { return nil }

func makeFS() store.FileStore { return store.FileStore{Fmt: stubFormat{}} }

// createTempDir creates a temp dir with the given regular files (base names).
// Returns the dir path and a cleanup function.
func createTempDir(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range files {
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte("---\ntitle: test\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFileStore_Enumerate_GlobHits(t *testing.T) {
	dir := createTempDir(t, "a.md", "b.md", "c.txt")
	pattern := filepath.Join(dir, "*.md")

	fs := makeFS()
	paths, err := fs.Enumerate([]string{pattern}, store.EnumOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestFileStore_Enumerate_HiddenFiltered(t *testing.T) {
	dir := createTempDir(t, ".hidden.md", "visible.md")
	pattern := filepath.Join(dir, "*.md")

	fs := makeFS()

	// Hidden filter ON (default)
	paths, err := fs.Enumerate([]string{pattern}, store.EnumOptions{IncludeHidden: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path after filtering, got %d: %v", len(paths), paths)
	}

	// Hidden filter OFF
	all, err := fs.Enumerate([]string{pattern}, store.EnumOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 paths with hidden included, got %d: %v", len(all), all)
	}
}

func TestFileStore_Enumerate_BarePath(t *testing.T) {
	dir := createTempDir(t, "note.md")
	bare := filepath.Join(dir, "note.md")

	fs := makeFS()
	paths, err := fs.Enumerate([]string{bare}, store.EnumOptions{IncludeHidden: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || paths[0] != bare {
		t.Fatalf("expected [%s], got %v", bare, paths)
	}
}

func TestFileStore_Enumerate_BarePathHiddenKept(t *testing.T) {
	dir := createTempDir(t, ".hidden.md")
	bare := filepath.Join(dir, ".hidden.md")

	fs := makeFS()
	// A bare (non-glob) path to a hidden file must be kept even with IncludeHidden=false.
	paths, err := fs.Enumerate([]string{bare}, store.EnumOptions{IncludeHidden: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected bare hidden path to be kept, got %v", paths)
	}
}

func TestFileStore_Enumerate_NonexistentError(t *testing.T) {
	fs := makeFS()
	_, err := fs.Enumerate([]string{"/nonexistent/path/to/file.md"}, store.EnumOptions{})
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}
