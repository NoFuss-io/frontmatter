package main

import (
	"testing"
)

// TestAlterArgs exercises the full pipeline from args through newAlterStatement
// and verifies the on-disk file state after the statement runs.
func TestAlterArgs(t *testing.T) {
	tests := []struct {
		name       string
		file       testFile
		args       func(path string) []string
		wantFM     map[string]string // fields that must be present with these formatted values
		wantAbsent []string          // fields that must be absent after the alter
		wantErr    bool
	}{
		{
			name: "drop removes existing field",
			file: tfDraftHello,
			args: func(path string) []string {
				return []string{path, "drop", "draft"}
			},
			wantFM:     map[string]string{"title": "Hello"},
			wantAbsent: []string{"draft"},
		},
		{
			name: "where clause drops field from matching file",
			file: tfDraftPublished,
			args: func(path string) []string {
				return []string{path, "drop", "draft", "where", "published=true"}
			},
			wantFM:     map[string]string{"published": "true"},
			wantAbsent: []string{"draft"},
		},
		{
			name: "where clause skips non-matching file",
			file: tfDraftNotPublished,
			args: func(path string) []string {
				return []string{path, "drop", "draft", "where", "published=true"}
			},
			wantFM: map[string]string{"draft": "true", "published": "false"},
		},
		{
			name: "type annotation drops field only when type matches",
			file: tfIntRatingPie,
			args: func(path string) []string {
				return []string{path, "drop", "rating:int"}
			},
			wantFM:     map[string]string{"title": "Pie"},
			wantAbsent: []string{"rating"},
		},
		{
			name: "type annotation leaves field when stored type differs",
			file: tfStrRatingPie,
			args: func(path string) []string {
				return []string{path, "drop", "rating:int"}
			},
			wantFM: map[string]string{"rating": "excellent", "title": "Pie"},
		},
		{
			name: "missing drop keyword returns parse error",
			args: func(path string) []string {
				return []string{"nonexistent.md", "draft"}
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var path string
			if tc.file.name != "" {
				path = writeTF(t, dir, tc.file)
			}
			stmt, err := newAlterStatement(tc.args(path))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error from newAlterStatement, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("newAlterStatement: %v", err)
			}
			if err := stmt.run(); err != nil {
				t.Fatalf("run: %v", err)
			}
			f, err := ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile after run: %v", err)
			}
			checkFM(t, f, tc.wantFM)
			checkFMAbsent(t, f, tc.wantAbsent...)
		})
	}
}

// TestAlterStatement constructs an alterStatement directly (bypassing arg parsing)
// and verifies the FM state of a single file after the statement runs.
func TestAlterStatement(t *testing.T) {
	tests := []struct {
		name       string
		file       testFile
		stmt       func(path string) alterStatement
		wantFM     map[string]string
		wantAbsent []string
	}{
		{
			name: "drops existing field",
			file: tfDraftHello,
			stmt: func(path string) alterStatement {
				return alterStatement{
					globs:  []string{path},
					fields: []Field{{Name: "draft"}},
				}
			},
			wantFM:     map[string]string{"title": "Hello"},
			wantAbsent: []string{"draft"},
		},
		{
			name: "drops field when where expression matches",
			file: tfDraftPublished,
			stmt: func(path string) alterStatement {
				expr, _ := ParseExpression("published=true")
				return alterStatement{
					globs:  []string{path},
					fields: []Field{{Name: "draft"}},
					where:  expr,
				}
			},
			wantFM:     map[string]string{"published": "true"},
			wantAbsent: []string{"draft"},
		},
		{
			name: "leaves field when where expression does not match",
			file: tfDraftNotPublished,
			stmt: func(path string) alterStatement {
				expr, _ := ParseExpression("published=true")
				return alterStatement{
					globs:  []string{path},
					fields: []Field{{Name: "draft"}},
					where:  expr,
				}
			},
			wantFM: map[string]string{"draft": "true", "published": "false"},
		},
		{
			name: "type-annotated drop removes field whose stored type matches",
			file: tfIntRatingPie,
			stmt: func(path string) alterStatement {
				return alterStatement{
					globs:  []string{path},
					fields: []Field{{Name: "rating", Type: TypeInt}},
				}
			},
			wantFM:     map[string]string{"title": "Pie"},
			wantAbsent: []string{"rating"},
		},
		{
			name: "type-annotated drop leaves field whose stored type differs",
			file: tfStrRatingPie,
			stmt: func(path string) alterStatement {
				return alterStatement{
					globs:  []string{path},
					fields: []Field{{Name: "rating", Type: TypeInt}},
				}
			},
			wantFM: map[string]string{"rating": "excellent", "title": "Pie"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeTF(t, dir, tc.file)
			if err := tc.stmt(path).run(); err != nil {
				t.Fatalf("run: %v", err)
			}
			f, err := ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile after run: %v", err)
			}
			checkFM(t, f, tc.wantFM)
			checkFMAbsent(t, f, tc.wantAbsent...)
		})
	}
}
