package main

import (
	"os"
	"path/filepath"
	"testing"
)

type testFile struct {
	name    string
	content string
}

func tempMD(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTF(t *testing.T, dir string, tf testFile) string {
	t.Helper()
	return tempMD(t, dir, tf.name, tf.content)
}

func writeTFs(t *testing.T, dir string, files []testFile) {
	t.Helper()
	for _, tf := range files {
		writeTF(t, dir, tf)
	}
}

// checkFM verifies each field in want matches the formatted value in f.FM.
// An empty string matches a nil value (fmtValue(nil) == "").
func checkFM(t *testing.T, f *File, want map[string]string) {
	t.Helper()
	for k, wantV := range want {
		gotV, ok := f.FM[k]
		if !ok {
			t.Errorf("FM[%q]: field absent, want %q", k, wantV)
			continue
		}
		if got := fmtValue(gotV); got != wantV {
			t.Errorf("FM[%q] = %q (raw %T %v), want %q", k, got, gotV, gotV, wantV)
		}
	}
}

func checkFMAbsent(t *testing.T, f *File, fields ...string) {
	t.Helper()
	for _, k := range fields {
		if v, ok := f.FM[k]; ok {
			t.Errorf("FM[%q] should be absent, got %v", k, v)
		}
	}
}

// Test files — treat as constants; reuse freely across tests.

var (
	// select: single-field output
	tfMinestrone = testFile{"soup.md", "---\ntitle: Minestrone\nrating: 4\n---\n"}

	// select: where-clause filtering (two files needed for the glob)
	tfPublishedA = testFile{"a.md", "---\ntitle: Published\npublished: true\n---\n"}
	tfDraftB     = testFile{"b.md", "---\ntitle: Draft\npublished: false\n---\n"}

	// select: sort by date
	tfOlder = testFile{"old.md", "---\ntitle: Older\ndate: 2024-01-01\n---\n"}
	tfNewer = testFile{"new.md", "---\ntitle: Newer\ndate: 2024-06-01\n---\n"}

	// select: limit (three files needed)
	tfAlpha = testFile{"a.md", "---\ntitle: Alpha\n---\n"}
	tfBeta  = testFile{"b.md", "---\ntitle: Beta\n---\n"}
	tfGamma = testFile{"c.md", "---\ntitle: Gamma\n---\n"}

	// select statement: single-file cases
	tfStew             = testFile{"test.md", "---\ntitle: Stew\ndate: 2024-03-10\n---\nBody.\n"}
	tfCakePublished    = testFile{"test.md", "---\ntitle: Cake\npublished: true\n---\n"}
	tfDraftUnpublished = testFile{"test.md", "---\ntitle: Draft\npublished: false\n---\n"}
	tfRatingExcellent  = testFile{"test.md", "---\nrating: excellent\n---\n"}
	tfNoDate           = testFile{"test.md", "---\ntitle: NoDates\n---\n"}

	// update args
	tfTitleOld          = testFile{"test.md", "---\ntitle: Old\n---\n"}
	tfCountFive         = testFile{"test.md", "---\ncount: 5\n---\n"}
	tfTagsBaking        = testFile{"test.md", "---\ntags:\n  - baking\n---\n"}
	tfDraftHello        = testFile{"test.md", "---\ndraft: true\ntitle: Hello\n---\n"} // also used by alter
	tfRecipeUnpublished = testFile{"test.md", "---\ntitle: Recipe\npublished: false\n---\n"}

	// update statement
	tfTitleOriginal    = testFile{"test.md", "---\ntitle: Original\n---\n"}
	tfScoreTen         = testFile{"test.md", "---\nscore: 10\n---\n"}
	tfTagsCooking      = testFile{"test.md", "---\ntags:\n  - cooking\n---\n"}
	tfTagsCookingVegan = testFile{"test.md", "---\ntags:\n  - cooking\n  - vegan\n---\n"}
	tfTagsThree        = testFile{"test.md", "---\ntags:\n  - cooking\n  - vegan\n  - quick\n---\n"}

	// alter (shared between TestAlterArgs and TestAlterStatement)
	tfDraftPublished    = testFile{"test.md", "---\ndraft: true\npublished: true\n---\n"}
	tfDraftNotPublished = testFile{"test.md", "---\ndraft: true\npublished: false\n---\n"}
	tfIntRatingPie      = testFile{"test.md", "---\nrating: 4\ntitle: Pie\n---\n"}
	tfStrRatingPie      = testFile{"test.md", "---\nrating: excellent\ntitle: Pie\n---\n"}
)
