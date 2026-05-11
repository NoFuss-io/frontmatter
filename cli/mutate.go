package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func writeErr(f *File, err error) {
	fmt.Fprintf(os.Stderr, "error writing %s: %v\n", f.Path, err)
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <glob>... set <assignment>[, <assignment>]... [where <expression>]",
		Short: "Cast or set field values",
		Long: `Applies one or more assignments to each matching file and writes the result back to
disk. Field order is always sorted alphabetically after a write.

Assignment forms:

  field:type      Cast field to type; creates it as null if absent. Skipped when type is any.
  field=value     Set field to value. Use null to clear the field.
  field+=value    Numbers: add. Strings: append. Lists: append if not already present.
  field-=value    Numbers: subtract. Lists: remove if present.`,
		Args: cobra.ArbitraryArgs,
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

			modified := 0
			for _, f := range files {
				if verbose >= 2 {
					fmt.Fprintln(os.Stderr, f.Path)
				}
				for _, a := range assignments {
					if err := f.Apply(a); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", f.Path, err)
					}
				}
				if err := f.Write(); err != nil {
					writeErr(f, err)
				} else {
					modified++
				}
			}
			if verbose >= 2 {
				fmt.Fprintf(os.Stderr, "%d files\n", len(files))
			} else if verbose >= 1 {
				fmt.Fprintf(os.Stderr, "%d files modified\n", modified)
			}
			return nil
		},
	}
}

func alterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alter <glob>... drop <field>[, <field>]... [where <expression>]",
		Short: "Remove fields",
		Long: `Removes one or more fields from each matching file's frontmatter and writes the
result back to disk. When a field carries a type annotation (field:type), it is only
removed if the stored value matches that type.`,
		Args: cobra.ArbitraryArgs,
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

			modified := 0
			for _, f := range files {
				if verbose >= 2 {
					fmt.Fprintln(os.Stderr, f.Path)
				}
				changed := false
				for _, field := range fields {
					if f.Remove(field) {
						changed = true
					}
				}
				if changed {
					if err := f.Write(); err != nil {
						writeErr(f, err)
					} else {
						modified++
					}
				}
			}
			if verbose >= 2 {
				fmt.Fprintf(os.Stderr, "%d files\n", len(files))
			} else if verbose >= 1 {
				fmt.Fprintf(os.Stderr, "%d files modified\n", modified)
			}
			return nil
		},
	}
}
