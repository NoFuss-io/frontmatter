package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var filterExpr string

// splitArgs separates file/glob args from field-spec args.
// Arg N is a file arg if it's a glob that expands to files, or an existing path.
// The first arg that matches neither starts the field-spec tail.
func splitArgs(args []string) (files []string, rest []string) {
	for i, arg := range args {
		if strings.ContainsAny(arg, "*?[") {
			if expanded, err := filepath.Glob(arg); err == nil && len(expanded) > 0 {
				files = append(files, expanded...)
				continue
			}
		}
		if _, err := os.Stat(arg); err == nil {
			files = append(files, arg)
			continue
		}
		rest = args[i:]
		return
	}
	return
}

func loadFiles(paths []string) ([]*File, error) {
	var expr Expression
	var err error
	if filterExpr != "" {
		expr, err = ParseExpression(filterExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %w", err)
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
	root.PersistentFlags().StringVar(&filterExpr, "filter", "", "filter expression")
	root.AddCommand(listCmd(), setCmd(), rmCmd(), castCmd(), checkCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <glob|files...> [<field|comparison>...]",
		Short: "Output table of filename and field values",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePaths, fieldArgs := splitArgs(args)
			if len(filePaths) == 0 {
				return fmt.Errorf("no files specified")
			}
			files, err := loadFiles(filePaths)
			if err != nil {
				return err
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

			if len(cols) == 0 {
				for _, f := range filtered {
					fmt.Println(f.Path)
				}
				return nil
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

func setCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <glob|files...> <field=value>...",
		Short: "Set field values",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePaths, fieldArgs := splitArgs(args)
			if len(filePaths) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(fieldArgs) == 0 {
				return fmt.Errorf("no field=value assignments specified")
			}
			files, err := loadFiles(filePaths)
			if err != nil {
				return err
			}
			var cmps []Comparison
			for _, arg := range fieldArgs {
				cmp, err := ParseComparison(arg)
				if err != nil {
					return err
				}
				if cmp.Value == nil {
					return fmt.Errorf("set requires field=value, got %q", arg)
				}
				cmps = append(cmps, cmp)
			}
			for _, f := range files {
				for _, cmp := range cmps {
					f.Set(cmp.Field.Name, *cmp.Value)
				}
				if err := f.Write(); err != nil {
					writeErr(f, err)
				}
			}
			return nil
		},
	}
}

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <glob|files...> <field>",
		Short: "Remove a field (only if type matches when type is specified)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePaths, fieldArgs := splitArgs(args)
			if len(filePaths) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(fieldArgs) != 1 {
				return fmt.Errorf("expected exactly one field, got %d", len(fieldArgs))
			}
			files, err := loadFiles(filePaths)
			if err != nil {
				return err
			}
			field, err := ParseField(fieldArgs[0])
			if err != nil {
				return err
			}
			for _, f := range files {
				if f.Remove(field) {
					if err := f.Write(); err != nil {
						writeErr(f, err)
					}
				}
			}
			return nil
		},
	}
}

func castCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cast <glob|files...> <field> <type>",
		Short: "Cast a field to a different type",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePaths, fieldArgs := splitArgs(args)
			if len(filePaths) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(fieldArgs) != 2 {
				return fmt.Errorf("expected <field> <type>, got %d args", len(fieldArgs))
			}
			files, err := loadFiles(filePaths)
			if err != nil {
				return err
			}
			field, err := ParseField(fieldArgs[0])
			if err != nil {
				return err
			}
			targetType, err := parseTypeName(fieldArgs[1])
			if err != nil {
				return err
			}
			for _, f := range files {
				if err := f.Cast(field, targetType); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", f.Path, err)
					continue
				}
				if err := f.Write(); err != nil {
					writeErr(f, err)
				}
			}
			return nil
		},
	}
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <glob|files...> <field>",
		Short: "Check that a field exists and matches its type",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePaths, fieldArgs := splitArgs(args)
			if len(filePaths) == 0 {
				return fmt.Errorf("no files specified")
			}
			if len(fieldArgs) != 1 {
				return fmt.Errorf("expected exactly one field, got %d", len(fieldArgs))
			}
			files, err := loadFiles(filePaths)
			if err != nil {
				return err
			}
			field, err := ParseField(fieldArgs[0])
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			anyFail := false
			for _, f := range files {
				status := "ok"
				if !f.CheckType(field) {
					status = "FAIL"
					anyFail = true
				}
				fmt.Fprintf(w, "%s\t%s\n", f.Path, status)
			}
			w.Flush()
			if anyFail {
				return fmt.Errorf("some files failed the check")
			}
			return nil
		},
	}
}
