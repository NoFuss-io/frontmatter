package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"text/tabwriter"

	"github.com/spf13/cobra"
)

type selectCol struct {
	field Field
	cmp   *Comparison
}

type SelectResult struct {
	headers []string
	rows    [][]string
}

func (r *SelectResult) addFile(f *File, cols []selectCol) {
	row := make([]string, 1+len(cols))
	row[0] = f.Path
	for i, c := range cols {
		if v, ok := f.FM[c.field.Name]; ok {
			row[i+1] = fmtValue(v)
		}
	}
	r.rows = append(r.rows, row)
}

func (r *SelectResult) print(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	seps := make([]string, len(r.headers))
	for i, h := range r.headers {
		seps[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(r.headers, "\t"))
	fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for _, row := range r.rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

func matchesCols(f *File, cols []selectCol) bool {
	for _, c := range cols {
		if c.cmp != nil && !f.matchesCmp(*c.cmp) {
			return false
		}
	}
	return true
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
		Args: cobra.ArbitraryArgs,
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
			sortFields, desc, err := parseSortClause(hasSortBy, sortRest)
			if err != nil {
				return err
			}
			limit, err := parseLimitClause(hasLimit, limitRest)
			if err != nil {
				return err
			}

			seq, err := iterFiles(paths, whereArgs)
			if err != nil {
				return err
			}

			if len(fieldArgs) == 0 {
				if hasSortBy {
					var files []*File
					for f := range seq {
						files = append(files, f)
					}
					sortFiles(files, sortFields, desc)
					for _, f := range applyLimit(files, limit) {
						fmt.Println(f.Path)
					}
				} else {
					count := 0
					for f := range seq {
						fmt.Println(f.Path)
						count++
						if limit > 0 && count >= limit {
							break
						}
					}
				}
				return nil
			}

			var cols []selectCol
			for _, arg := range fieldArgs {
				cmp, err := ParseComparison(arg)
				if err != nil {
					return err
				}
				c := selectCol{field: cmp.Field}
				if cmp.Value != nil {
					c.cmp = &cmp
				}
				cols = append(cols, c)
			}

			headers := make([]string, 1+len(cols))
			headers[0] = "filename"
			for i, c := range cols {
				headers[i+1] = c.field.Name
			}
			result := &SelectResult{headers: headers}

			if hasSortBy {
				var files []*File
				for f := range seq {
					if matchesCols(f, cols) {
						files = append(files, f)
					}
				}
				sortFiles(files, sortFields, desc)
				for _, f := range applyLimit(files, limit) {
					result.addFile(f, cols)
				}
			} else {
				for f := range seq {
					if matchesCols(f, cols) {
						result.addFile(f, cols)
					}
					if limit > 0 && len(result.rows) >= limit {
						break
					}
				}
			}

			result.print(os.Stdout)
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
