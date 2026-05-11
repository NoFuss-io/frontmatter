package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

const VERSION = "v0.1" // TODO: Repalce with semver

// splitOn splits args on the first occurrence of keyword, returning the two halves.
func splitOn(args []string, keyword string) (before, after []string, found bool) {
	for i, a := range args {
		if a == keyword {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
}

// splitCommas expands comma-separated tokens, handling "a,b", "a, b", and "a , b".
func splitCommas(args []string) []string {
	var out []string
	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

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

func writeErr(f *File, err error) {
	fmt.Fprintf(os.Stderr, "error writing %s: %v\n", f.Path, err)
}

func main() {
	root := &cobra.Command{
		Use:          "fm",
		Short:        fmt.Sprintf("Markdown frontmatter batch editor (%s)", VERSION),
		SilenceUsage: true,
	}
	root.AddCommand(selectCmd(), updateCmd(), alterCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func selectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <field>[, <field>]... from <glob>... [where <expression>] [sort by <field>[, <field>]... [desc]] [limit <n>]",
		Short: "Output table of filename and field values",
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

func parseSortClause(present bool, tokens []string) ([]Field, bool, error) {
	if !present {
		return nil, false, nil
	}
	if len(tokens) == 0 || tokens[0] != "by" {
		return nil, false, fmt.Errorf("expected 'by' after 'sort'")
	}
	args := splitCommas(tokens[1:])
	desc := len(args) > 0 && args[len(args)-1] == "desc"
	if desc {
		args = args[:len(args)-1]
	}
	if len(args) == 0 {
		return nil, false, fmt.Errorf("no fields specified after 'sort by'")
	}
	var fields []Field
	for _, arg := range args {
		f, err := ParseField(arg)
		if err != nil {
			return nil, false, err
		}
		fields = append(fields, f)
	}
	return fields, desc, nil
}

func parseLimitClause(present bool, tokens []string) (int, error) {
	if !present {
		return 0, nil
	}
	if len(tokens) != 1 {
		return 0, fmt.Errorf("limit requires exactly one integer argument")
	}
	n, err := strconv.Atoi(tokens[0])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("limit must be a non-negative integer, got %q", tokens[0])
	}
	return n, nil
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

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <glob>... set <assignment>[, <assignment>]... [where <expression>]",
		Short: "Cast or set field values",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			globArgs, rest, ok := splitOn(args, "set")
			if !ok {
				return fmt.Errorf("missing 'set' keyword")
			}
			assignArgs, whereArgs, _ := splitOn(rest, "where")
			assignArgs = splitCommas(assignArgs)

			if len(globArgs) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(assignArgs) == 0 {
				return fmt.Errorf("no fields or assignments specified")
			}
			paths, err := expandGlobs(globArgs)
			if err != nil {
				return err
			}
			files, err := loadFiles(paths, whereArgs)
			if err != nil {
				return err
			}

			var assignments []Assignment
			for _, arg := range assignArgs {
				a, err := ParseAssignment(arg)
				if err != nil {
					return err
				}
				assignments = append(assignments, a)
			}

			for _, f := range files {
				for _, a := range assignments {
					if err := f.Apply(a); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", f.Path, err)
					}
				}
				if err := f.Write(); err != nil {
					writeErr(f, err)
				}
			}
			return nil
		},
	}
}

func alterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alter <glob>... drop <field>[, <field>]... [where <expression>]",
		Short: "Remove fields",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			globArgs, rest, ok := splitOn(args, "drop")
			if !ok {
				return fmt.Errorf("missing 'drop' keyword")
			}
			fieldArgs, whereArgs, _ := splitOn(rest, "where")
			fieldArgs = splitCommas(fieldArgs)

			if len(globArgs) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(fieldArgs) == 0 {
				return fmt.Errorf("no fields specified")
			}
			paths, err := expandGlobs(globArgs)
			if err != nil {
				return err
			}
			files, err := loadFiles(paths, whereArgs)
			if err != nil {
				return err
			}

			var fields []Field
			for _, arg := range fieldArgs {
				field, err := ParseField(arg)
				if err != nil {
					return err
				}
				fields = append(fields, field)
			}

			for _, f := range files {
				changed := false
				for _, field := range fields {
					if f.Remove(field) {
						changed = true
					}
				}
				if changed {
					if err := f.Write(); err != nil {
						writeErr(f, err)
					}
				}
			}
			return nil
		},
	}
}
