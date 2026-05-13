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
				for _, path := range result.files {
					fmt.Println(path)
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
	paths, err := expandGlobs(s.globs)
	if err != nil {
		return nil, err
	}
	n := len(paths)

	// No sorting -> Limit before processing
	if n > s.limit && s.limit > 0 && len(s.sortFields) == 0 {
		paths = paths[:n]
	}

	headers := make([]string, len(s.cols))
	for i, c := range s.cols {
		headers[i] = c.field.Name
	}
	result := &SelectResult{headers: headers}

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

	// Sorting -> Limit after processing and sorting
	if len(s.sortFields) > 0 {
		sortFiles(files, s.sortFields, s.sortDesc)
		if n > s.limit && s.limit > 0 {
			files = files[:n]
		}
	}

	for _, f := range files {
		result.addRow(f, s.cols)
	}

	return result, nil
}

type selectCol struct {
	field Field
	cmp   *Comparison
}

type SelectResult struct {
	files   []string   // one path per row
	headers []string   // field column headers (excludes filename)
	rows    [][]string // field values per row (excludes filename)
}

func (r *SelectResult) addRow(f *File, cols []selectCol) {
	r.files = append(r.files, f.Path)
	row := make([]string, len(cols))
	for i, c := range cols {
		if v, ok := f.FM[c.field.Name]; ok {
			row[i] = fmtValue(v)
		}
	}
	r.rows = append(r.rows, row)
}

func (r *SelectResult) print(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	headers := append([]string{"filename"}, r.headers...)
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for i, row := range r.rows {
		fmt.Fprintln(tw, strings.Join(append([]string{r.files[i]}, row...), "\t"))
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
