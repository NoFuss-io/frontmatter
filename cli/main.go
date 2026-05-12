package main

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type Semver struct{ Major, Minor, Patch int }

func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

var VERSION = Semver{0, 1, 0}

var verbose int

func expandGlobs(patterns []string) ([]string, error) {
	var files []string
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?[") {
			expanded, err := filepath.Glob(p)
			if err != nil {
				return nil, err
			}
			files = append(files, expanded...)
		} else if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		} else {
			return nil, fmt.Errorf("no such file or pattern: %s", p)
		}
	}
	return files, nil
}

func iterFiles(paths []string, whereArgs []string) (iter.Seq[*File], error) {
	var expr Expression
	if len(whereArgs) > 0 {
		var err error
		expr, err = ParseExpression(strings.Join(whereArgs, " "))
		if err != nil {
			return nil, fmt.Errorf("invalid where clause: %w", err)
		}
	}
	return func(yield func(*File) bool) {
		for _, path := range paths {
			f, err := ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				continue
			}
			if f.Matches(expr) && !yield(f) {
				return
			}
		}
	}, nil
}

func main() {
	root := &cobra.Command{
		Use:   "fm",
		Short: fmt.Sprintf("Markdown frontmatter batch editor (%s)", VERSION),
		Long: `fm is a command-line tool for batch-querying and editing YAML frontmatter in
Markdown files. It is designed for knowledge management workflows such as Obsidian
vaults, where structured metadata lives in a --- YAML block at the top of each .md file.

Commands follow a SQL-inspired syntax:

  fm select <fields> from <files> [where <expr>] [sort by <field> [desc]] [limit <n>]
  fm update <files> set <assignments> [where <expr>]
  fm alter  <files> drop <fields> [where <expr>]`,
		SilenceUsage: true,
	}
	root.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Verbosity: -v prints modified count, -vv prints each processed file and total")
	root.AddCommand(selectCmd(), updateCmd(), alterCmd())
	root.AddCommand(genManCmd(root), installManCmd(root))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func writeErr(f *File, err error) {
	fmt.Fprintf(os.Stderr, "error writing %s: %v\n", f.Path, err)
}

