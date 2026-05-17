package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestSelectOutput exercises the full pipeline from args through newSelectStatement
// and result.print, checking that the rendered table text contains expected values.
func TestSelectOutput(t *testing.T) {
	tests := []struct {
		name       string
		files      []testFile
		args       func(dir string) []string
		wantOut    []string  // substrings that must appear in output
		wantAbsent []string  // substrings that must not appear in output
		wantOrder  [2]string // if both non-empty, wantOrder[0] must precede wantOrder[1]
		wantErr    bool
	}{
		{
			name:  "single field from one file",
			files: []testFile{tfMinestrone},
			args: func(dir string) []string {
				return []string{"title", "from", filepath.Join(dir, "soup.md")}
			},
			wantOut: []string{"filename", "title", "soup.md", "Minestrone"},
		},
		{
			name:  "where clause keeps matching file and excludes other",
			files: []testFile{tfPublishedA, tfDraftB},
			args: func(dir string) []string {
				return []string{"title", "from", filepath.Join(dir, "*.md"), "where", "published=true"}
			},
			wantOut:    []string{"Published"},
			wantAbsent: []string{"Draft"},
		},
		{
			name:  "sort by date descending places newest row first",
			files: []testFile{tfOlder, tfNewer},
			args: func(dir string) []string {
				return []string{"title,", "date", "from", filepath.Join(dir, "*.md"), "sort", "by", "date", "desc"}
			},
			wantOut:   []string{"Newer", "Older"},
			wantOrder: [2]string{"Newer", "Older"},
		},
		{
			name:  "limit truncates output after sort",
			files: []testFile{tfAlpha, tfBeta, tfGamma},
			args: func(dir string) []string {
				return []string{"title", "from", filepath.Join(dir, "*.md"), "sort", "by", "title", "limit", "2"}
			},
			wantOut:    []string{"Alpha", "Beta"},
			wantAbsent: []string{"Gamma"},
		},
		{
			name: "missing from keyword returns parse error",
			args: func(dir string) []string {
				return []string{"title", "soup.md"}
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTFs(t, dir, tc.files)
			stmt, err := newSelectStatement(tc.args(dir))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error from newSelectStatement, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("newSelectStatement: %v", err)
			}
			result, err := stmt.run()
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			var buf bytes.Buffer
			result.print(&buf)
			out := buf.String()
			for _, want := range tc.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, out)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("output should not contain %q\nfull output:\n%s", absent, out)
				}
			}
			if tc.wantOrder[0] != "" && tc.wantOrder[1] != "" {
				i := strings.Index(out, tc.wantOrder[0])
				j := strings.Index(out, tc.wantOrder[1])
				if i < 0 || j < 0 || i >= j {
					t.Errorf("expected %q before %q in output\nfull output:\n%s",
						tc.wantOrder[0], tc.wantOrder[1], out)
				}
			}
		})
	}
}

// TestSelectStatement constructs a selectStatement directly (bypassing arg parsing)
// and verifies the returned SelectResult for a single file.
func TestSelectStatement(t *testing.T) {
	tests := []struct {
		name      string
		file      testFile
		stmt      func(path string) selectStatement
		wantFiles int
		wantRow   []string // formatted values in result.rows[0]; nil skips row check
	}{
		{
			name: "returns title and date for matching file",
			file: tfStew,
			stmt: func(path string) selectStatement {
				return selectStatement{
					cols:  []selectCol{{field: Field{Name: "title"}}, {field: Field{Name: "date"}}},
					globs: []string{path},
				}
			},
			wantFiles: 1,
			wantRow:   []string{"Stew", "2024-03-10"},
		},
		{
			name: "where expression matches file",
			file: tfCakePublished,
			stmt: func(path string) selectStatement {
				expr, _ := ParseExpression("published=true")
				return selectStatement{
					cols:  []selectCol{{field: Field{Name: "title"}}},
					globs: []string{path},
					where: expr,
				}
			},
			wantFiles: 1,
			wantRow:   []string{"Cake"},
		},
		{
			name: "where expression excludes non-matching file",
			file: tfDraftUnpublished,
			stmt: func(path string) selectStatement {
				expr, _ := ParseExpression("published=true")
				return selectStatement{
					cols:  []selectCol{{field: Field{Name: "title"}}},
					globs: []string{path},
					where: expr,
				}
			},
			wantFiles: 0,
		},
		{
			name: "col with type comparison excludes file where field has wrong type",
			file: tfRatingExcellent,
			stmt: func(path string) selectStatement {
				// Simulate a type-annotated column that filters rows where the field
				// doesn't hold the expected type (rating:int excludes a string value).
				cmp := Comparison{Field: Field{Name: "rating", Type: TypeInt}}
				return selectStatement{
					cols:  []selectCol{{field: Field{Name: "rating", Type: TypeInt}, cmp: &cmp}},
					globs: []string{path},
				}
			},
			wantFiles: 0,
		},
		{
			name: "missing field renders as empty cell",
			file: tfNoDate,
			stmt: func(path string) selectStatement {
				return selectStatement{
					cols:  []selectCol{{field: Field{Name: "title"}}, {field: Field{Name: "date"}}},
					globs: []string{path},
				}
			},
			wantFiles: 1,
			wantRow:   []string{"NoDates", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeTF(t, dir, tc.file)
			result, err := tc.stmt(path).run()
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if len(result.files) != tc.wantFiles {
				t.Errorf("got %d result rows, want %d", len(result.files), tc.wantFiles)
			}
			if tc.wantRow != nil && len(result.rows) > 0 {
				row := result.rows[0]
				for i, want := range tc.wantRow {
					if i >= len(row) {
						t.Errorf("row[%d] missing, want %q", i, want)
					} else if row[i] != want {
						t.Errorf("row[%d] = %q, want %q", i, row[i], want)
					}
				}
			}
		})
	}
}
