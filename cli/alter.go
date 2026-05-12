package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

			var fields []Field
			for _, arg := range fieldArgs {
				field, err := ParseField(arg)
				if err != nil {
					return err
				}
				fields = append(fields, field)
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
				changed := false
				for _, field := range fields {
					if f.Remove(field) {
						changed = true
					}
				}
				if changed {
					if e := f.Write(); e != nil {
						writeErr(f, e)
					} else {
						modified++
					}
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
