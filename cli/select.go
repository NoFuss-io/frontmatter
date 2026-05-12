package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

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
			stmt, err := newSelectStatement(args)
			if err != nil {
				return err
			}
			result, err := stmt.run()
			if err != nil {
				return err
			}
			if len(stmt.cols) == 0 {
				for _, row := range result.rows {
					fmt.Println(row[0])
				}
			} else {
				result.print(os.Stdout)
			}
			return nil
		},
	}
}

type selectStatement struct {
	cols       []selectCol
	globs      []string
	where      Expression
	sortFields []Field
	sortDesc   bool
	limit      int
}

func newSelectStatement(args []string) (selectStatement, error) {
	fieldArgs, rest, _ := splitOn(args, "from")
	fieldArgs = splitCommas(fieldArgs)
	rest, limitRest, hasLimit := splitOn(rest, "limit")
	rest, sortRest, hasSortBy := splitOn(rest, "sort")
	globArgs, whereArgs, _ := splitOn(rest, "where")

	if len(globArgs) == 0 {
		return selectStatement{}, fmt.Errorf("no files specified (missing 'from'?)")
	}

	var cols []selectCol
	for _, arg := range fieldArgs {
		cmp, err := ParseComparison(arg)
		if err != nil {
			return selectStatement{}, err
		}
		c := selectCol{field: cmp.Field}
		if cmp.Value != nil {
			c.cmp = &cmp
		}
		cols = append(cols, c)
	}

	var where Expression
	if len(whereArgs) > 0 {
		var err error
		where, err = ParseExpression(strings.Join(whereArgs, " "))
		if err != nil {
			return selectStatement{}, fmt.Errorf("invalid where clause: %w", err)
		}
	}

	sortFields, sortDesc, err := parseSortClause(hasSortBy, sortRest)
	if err != nil {
		return selectStatement{}, err
	}

	limit, err := parseLimitClause(hasLimit, limitRest)
	if err != nil {
		return selectStatement{}, err
	}

	return selectStatement{
		cols:       cols,
		globs:      globArgs,
		where:      where,
		sortFields: sortFields,
		sortDesc:   sortDesc,
		limit:      limit,
	}, nil
}

func (s selectStatement) run() (*SelectResult, error) {
	var paths []string
	for _, p := range s.globs {
		if strings.ContainsAny(p, "*?[") {
			expanded, err := filepath.Glob(p)
			if err != nil {
				return nil, err
			}
			paths = append(paths, expanded...)
		} else if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		} else {
			return nil, fmt.Errorf("no such file or pattern: %s", p)
		}
	}

	headers := make([]string, 1+len(s.cols))
	headers[0] = "filename"
	for i, c := range s.cols {
		headers[i+1] = c.field.Name
	}
	result := &SelectResult{headers: headers}

	if len(s.sortFields) > 0 {
		var files []*File
		for _, path := range paths {
			f, err := ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				continue
			}
			if f.Matches(s.where) && matchesCols(f, s.cols) {
				files = append(files, f)
			}
		}
		sortFiles(files, s.sortFields, s.sortDesc)
		for _, f := range applyLimit(files, s.limit) {
			result.addFile(f, s.cols)
		}
	} else {
		for _, path := range paths {
			f, err := ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
				continue
			}
			if f.Matches(s.where) && matchesCols(f, s.cols) {
				result.addFile(f, s.cols)
			}
			if s.limit > 0 && len(result.rows) >= s.limit {
				break
			}
		}
	}

	return result, nil
}

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
