package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

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

func loadFiles(paths []string, whereArgs []string) ([]*File, error) {
	var expr Expression
	if len(whereArgs) > 0 {
		var err error
		expr, err = ParseExpression(strings.Join(whereArgs, " "))
		if err != nil {
			return nil, fmt.Errorf("invalid where clause: %w", err)
		}
	}
	var files []*File
	for _, path := range paths {
		f, err := ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			continue
		}
		if f.Matches(expr) {
			files = append(files, f)
		}
	}
	return files, nil
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

func selectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <field>[, <field>]... from <glob>... [where <expression>] [sort by <field>[, <field>]... [desc]] [limit <n>]",
		Short: "Output table of filename and field values",
		Long: `Prints a table with one row per matching file and one column per requested field.
With no field list, only filenames are printed.

Fields may carry a type annotation (field:type) to restrict output to files where that
field holds a value of the given type.

sort by accepts one or more comma-separated fields followed by an optional desc suffix
for descending order. Sorting is lexicographic; files missing the sort field sort last.
Dates stored as YYYY-MM-DD sort correctly as strings.

limit n truncates the output to at most n rows, applied after sorting.`,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldArgs, rest, _ := splitOn(args, "from")
			fieldArgs = splitCommas(fieldArgs)
			rest, limitRest, hasLimit := splitOn(rest, "limit")
			rest, sortRest, hasSortBy := splitOn(rest, "sort")
			globArgs, whereArgs, _ := splitOn(rest, "where")

			if len(globArgs) == 0 {
				return fmt.Errorf("no files specified (missing 'from'?)")
			}
			paths, err := expandGlobs(globArgs)
			if err != nil {
				return err
			}
			files, err := loadFiles(paths, whereArgs)
			if err != nil {
				return err
			}

			sortFields, desc, err := parseSortClause(hasSortBy, sortRest)
			if err != nil {
				return err
			}
			limit, err := parseLimitClause(hasLimit, limitRest)
			if err != nil {
				return err
			}

			if len(fieldArgs) == 0 {
				sortFiles(files, sortFields, desc)
				files = applyLimit(files, limit)
				for _, f := range files {
					fmt.Println(f.Path)
				}
				return nil
			}

			type col struct {
				field Field
				cmp   *Comparison
			}
			var cols []col
			for _, arg := range fieldArgs {
				cmp, err := ParseComparison(arg)
				if err != nil {
					return err
				}
				c := col{field: cmp.Field}
				if cmp.Value != nil {
					c.cmp = &cmp
				}
				cols = append(cols, c)
			}

			var filtered []*File
			for _, f := range files {
				match := true
				for _, c := range cols {
					if c.cmp != nil && !f.matchesCmp(*c.cmp) {
						match = false
						break
					}
				}
				if match {
					filtered = append(filtered, f)
				}
			}

			sortFiles(filtered, sortFields, desc)
			filtered = applyLimit(filtered, limit)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			headers := []string{"filename"}
			seps := []string{strings.Repeat("-", 8)}
			for _, c := range cols {
				headers = append(headers, c.field.Name)
				seps = append(seps, strings.Repeat("-", len(c.field.Name)))
			}
			fmt.Fprintln(w, strings.Join(headers, "\t"))
			fmt.Fprintln(w, strings.Join(seps, "\t"))
			for _, f := range filtered {
				row := []string{f.Path}
				for _, c := range cols {
					v, ok := f.FM[c.field.Name]
					if ok {
						row = append(row, fmtValue(v))
					} else {
						row = append(row, "")
					}
				}
				fmt.Fprintln(w, strings.Join(row, "\t"))
			}
			w.Flush()
			return nil
		},
	}
}

func applyLimit(files []*File, n int) []*File {
	if n == 0 || n >= len(files) {
		return files
	}
	return files[:n]
}

func sortFiles(files []*File, fields []Field, desc bool) {
	if len(fields) == 0 {
		return
	}
	sort.SliceStable(files, func(i, j int) bool {
		for _, sf := range fields {
			vi := fmtValue(files[i].FM[sf.Name])
			vj := fmtValue(files[j].FM[sf.Name])
			if vi == vj {
				continue
			}
			if desc {
				return vi > vj
			}
			return vi < vj
		}
		return false
	})
}

