package lib

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExpandGlobs(t *testing.T) {
	tutorialDir := filepath.Join(testDir(), "docs", "tutorial")
	if _, err := os.Stat(tutorialDir); os.IsNotExist(err) {
		t.Skipf("test fixture %s not found", tutorialDir)
	}

	tests := []struct {
		name    string
		globs   []string
		want    []string
		wantErr bool
	}{
		{
			name:  "bare path exists",
			globs: []string{filepath.Join(tutorialDir, "Apfelstrudel.md")},
			want:  []string{filepath.Join(tutorialDir, "Apfelstrudel.md")},
		},
		{
			name:    "bare path not exists",
			globs:   []string{filepath.Join(tutorialDir, "Nonexistent.md")},
			wantErr: true,
		},
		{
			name:  "glob with star",
			globs: []string{filepath.Join(tutorialDir, "*.md")},
			want: []string{
				filepath.Join(tutorialDir, "Apfelstrudel.md"),
				filepath.Join(tutorialDir, "Amaretto Cream.md"),
				filepath.Join(tutorialDir, "Baba Ganoush.md"),
				filepath.Join(tutorialDir, "Banana and Oat Cookies.md"),
				filepath.Join(tutorialDir, "Bacon-Wrapped Pork Tenderloin with Herbs and Mustard Sauce.md"),
				filepath.Join(tutorialDir, "Bacon Jam.md"),
				filepath.Join(tutorialDir, "Basil Chicken.md"),
				filepath.Join(tutorialDir, "Bechamel Sauce.md"),
				filepath.Join(tutorialDir, "Beluga Bolognese.md"),
				filepath.Join(tutorialDir, "Orange Cake.md"),
			},
		},
		{
			name:  "glob with no matches",
			globs: []string{filepath.Join(tutorialDir, "*.txt")},
			want:  []string{},
		},
		{
			name:  "multiple bare paths",
			globs: []string{filepath.Join(tutorialDir, "Apfelstrudel.md"), filepath.Join(tutorialDir, "Bacon Jam.md")},
			want: []string{
				filepath.Join(tutorialDir, "Apfelstrudel.md"),
				filepath.Join(tutorialDir, "Bacon Jam.md"),
			},
		},
		{
			name:  "glob prefix",
			globs: []string{filepath.Join(tutorialDir, "B*.md")},
			want: []string{
				filepath.Join(tutorialDir, "Baba Ganoush.md"),
				filepath.Join(tutorialDir, "Banana and Oat Cookies.md"),
				filepath.Join(tutorialDir, "Bacon-Wrapped Pork Tenderloin with Herbs and Mustard Sauce.md"),
				filepath.Join(tutorialDir, "Bacon Jam.md"),
				filepath.Join(tutorialDir, "Basil Chicken.md"),
				filepath.Join(tutorialDir, "Bechamel Sauce.md"),
				filepath.Join(tutorialDir, "Beluga Bolognese.md"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandGlobs(tt.globs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandGlobs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			slices.Sort(got)
			slices.Sort(tt.want)
			if !slices.Equal(got, tt.want) {
				t.Errorf("ExpandGlobs() got %v, want %v", got, tt.want)
			}
		})
	}
}

func testDir() string {
	// Walk up from lib/ to project root
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "lib" {
		return filepath.Dir(wd)
	}
	return "."
}
