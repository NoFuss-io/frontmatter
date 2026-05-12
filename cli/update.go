package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

			var assignments []Assignment
			for _, arg := range assignArgs {
				a, err := ParseAssignment(arg)
				if err != nil {
					return err
				}
				assignments = append(assignments, a)
			}

			seq, err := iterFiles(paths, whereArgs)
			if err != nil {
				return err
			}
			total, modified := 0, 0
			for f := range seq {
				total++
				if verbose >= 2 {
					fmt.Fprintln(os.Stderr, f.Path)
				}
				for _, a := range assignments {
					if e := f.Apply(a); e != nil {
						fmt.Fprintf(os.Stderr, "%s: %v\n", f.Path, e)
					}
				}
				if e := f.Write(); e != nil {
					writeErr(f, e)
				} else {
					modified++
				}
			}
			if verbose >= 2 {
				fmt.Fprintf(os.Stderr, "%d files\n", total)
			} else if verbose >= 1 {
				fmt.Fprintf(os.Stderr, "%d files modified\n", modified)
			}
			return nil
		},
	}
}

