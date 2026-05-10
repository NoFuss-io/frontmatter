package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// splitOn splits args on the first occurrence of keyword, returning the two halves.
func splitOn(args []string, keyword string) (before, after []string, found bool) {
	for i, a := range args {
		if a == keyword {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
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
		Short:        "Markdown frontmatter batch editor",
		SilenceUsage: true,
	}
	root.AddCommand(selectCmd(), updateCmd(), alterCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func selectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <field>... from <glob>... [where <expression>]",
		Short: "Output table of filename and field values",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldArgs, rest, _ := splitOn(args, "from")
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

			if len(fieldArgs) == 0 {
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

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <glob>... set <field|assignment>... [where <expression>]",
		Short: "Cast or set field values",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			globArgs, rest, ok := splitOn(args, "set")
			if !ok {
				return fmt.Errorf("missing 'set' keyword")
			}
			assignArgs, whereArgs, _ := splitOn(rest, "where")

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
		Use:   "alter <glob>... drop <field>... [where <expression>]",
		Short: "Remove fields",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			globArgs, rest, ok := splitOn(args, "drop")
			if !ok {
				return fmt.Errorf("missing 'drop' keyword")
			}
			fieldArgs, whereArgs, _ := splitOn(rest, "where")

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
