package main

import (
	"testing"
)

// TestUpdateArgs exercises the full pipeline from args through newUpdateStatement
// and verifies the on-disk file state after the statement runs.
func TestUpdateArgs(t *testing.T) {
	tests := []struct {
		name       string
		file       testFile
		args       func(path string) []string
		wantFM     map[string]string // field -> fmtValue(expected); "" matches nil
		wantAbsent []string          // fields that must be absent from FM
		wantErr    bool
	}{
		{
			name: "set replaces field value",
			file: tfTitleOld,
			args: func(path string) []string {
				return []string{path, "set", "title=NewTitle"}
			},
			wantFM: map[string]string{"title": "NewTitle"},
		},
		{
			name: "add increments integer field",
			file: tfCountFive,
			args: func(path string) []string {
				return []string{path, "set", "count+=3"}
			},
			wantFM: map[string]string{"count": "8"},
		},
		{
			name: "add appends to list field",
			file: tfTagsBaking,
			args: func(path string) []string {
				return []string{path, "set", "tags+=cooking"}
			},
			wantFM: map[string]string{"tags": "[baking, cooking]"},
		},
		{
			name: "set null clears field to nil",
			file: tfDraftHello,
			args: func(path string) []string {
				return []string{path, "set", "draft=null"}
			},
			wantFM: map[string]string{"draft": "", "title": "Hello"},
		},
		{
			name: "where clause skips non-matching file",
			file: tfRecipeUnpublished,
			args: func(path string) []string {
				return []string{path, "set", "title=Modified", "where", "published=true"}
			},
			wantFM: map[string]string{"title": "Recipe", "published": "false"},
		},
		{
			name: "missing set keyword returns parse error",
			args: func(path string) []string {
				return []string{"nonexistent.md", "title=New"}
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
			stmt, err := newUpdateStatement(tc.args(path))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error from newUpdateStatement, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("newUpdateStatement: %v", err)
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

// TestUpdateStatement constructs an updateStatement directly (bypassing arg parsing)
// and verifies the in-memory FM state of a single file after the statement runs.
func TestUpdateStatement(t *testing.T) {
	tests := []struct {
		name       string
		file       testFile
		stmt       func(path string) updateStatement
		wantFM     map[string]string
		wantAbsent []string
	}{
		{
			name: "set creates new field",
			file: tfTitleOriginal,
			stmt: func(path string) updateStatement {
				v := "Rewritten"
				return updateStatement{
					globs:       []string{path},
					assignments: []Assignment{{Field: Field{Name: "title"}, Op: OpSet, Value: &v}},
				}
			},
			wantFM: map[string]string{"title": "Rewritten"},
		},
		{
			name: "add to int accumulates value",
			file: tfScoreTen,
			stmt: func(path string) updateStatement {
				v := "5"
				return updateStatement{
					globs:       []string{path},
					assignments: []Assignment{{Field: Field{Name: "score"}, Op: OpAdd, Value: &v}},
				}
			},
			wantFM: map[string]string{"score": "15"},
		},
		{
			name: "add to list appends new item",
			file: tfTagsCooking,
			stmt: func(path string) updateStatement {
				v := "vegan"
				return updateStatement{
					globs:       []string{path},
					assignments: []Assignment{{Field: Field{Name: "tags"}, Op: OpAdd, Value: &v}},
				}
			},
			wantFM: map[string]string{"tags": "[cooking, vegan]"},
		},
		{
			name: "add to list is idempotent when item already present",
			file: tfTagsCookingVegan,
			stmt: func(path string) updateStatement {
				v := "vegan"
				return updateStatement{
					globs:       []string{path},
					assignments: []Assignment{{Field: Field{Name: "tags"}, Op: OpAdd, Value: &v}},
				}
			},
			wantFM: map[string]string{"tags": "[cooking, vegan]"},
		},
		{
			name: "subtract from list removes matching item",
			file: tfTagsThree,
			stmt: func(path string) updateStatement {
				v := "vegan"
				return updateStatement{
					globs:       []string{path},
					assignments: []Assignment{{Field: Field{Name: "tags"}, Op: OpSub, Value: &v}},
				}
			},
			wantFM: map[string]string{"tags": "[cooking, quick]"},
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
