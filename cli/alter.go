package main

import (
	"fmt"
	"os"
	"strings"

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
			stmt, err := newAlterStatement(args)
			if err != nil {
				return err
			}
			return stmt.run()
		},
	}
}

type alterStatement struct {
	globs  []string
	where  Expression
	fields []Field
}

func newAlterStatement(args []string) (alterStatement, error) {
	globArgs, rest, ok := splitOn(args, "drop")
	if !ok {
		return alterStatement{}, fmt.Errorf("missing 'drop' keyword")
	}
	fieldArgs, whereArgs, _ := splitOn(rest, "where")
	fieldArgs = splitCommas(fieldArgs)

	if len(globArgs) == 0 {
		return alterStatement{}, fmt.Errorf("no files specified")
	}
	if len(fieldArgs) == 0 {
		return alterStatement{}, fmt.Errorf("no fields specified")
	}

	var fields []Field
	for _, arg := range fieldArgs {
		field, err := ParseField(arg)
		if err != nil {
			return alterStatement{}, err
		}
		fields = append(fields, field)
	}

	var where Expression
	if len(whereArgs) > 0 {
		var err error
		where, err = ParseExpression(strings.Join(whereArgs, " "))
		if err != nil {
			return alterStatement{}, fmt.Errorf("invalid where clause: %w", err)
		}
	}

	return alterStatement{
		globs:  globArgs,
		where:  where,
		fields: fields,
	}, nil
}

func (s alterStatement) run() error {
	paths, err := expandGlobs(s.globs)
	if err != nil {
		return err
	}

	total, modified := 0, 0
	for _, path := range paths {
		f, err := ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			continue
		}
		if !f.Matches(s.where) {
			continue
		}
		total++
		if verbose >= 2 {
			fmt.Fprintln(os.Stderr, f.Path)
		}
		changed := false
		for _, field := range s.fields {
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
}
