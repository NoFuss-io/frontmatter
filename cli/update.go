package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update <glob>... set <assignment>[, <assignment>]... [where <expression>]",
		Short: "Cast or set field values",
		Long: `Applies one or more assignments to each matching file and writes the result back to
disk. Field order is always sorted alphabetically after a write.

Assignment forms:

  field:type        Cast field to type; creates it as null if absent. Skipped when type is any.
  field="string"    Set field to a literal string value.
  field=value       Set field to the value of another field (field reference).
  field=null        Clear the field (set to null).
  field+=value      Numbers: add. Strings: append. Lists: append if not already present.
  field+="value"    Lists: append literal string if not already present.
  field+=ref        Lists: merge (set union).
  field-="value"    Lists: remove literal string if present.
  field-=ref        Lists: subtract (set difference).

A missing field reference defaults to null for = and is a no-op for += and -=.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stmt, err := newUpdateStatement(args)
			if err != nil {
				return err
			}
			return stmt.run()
		},
	}
}

type updateStatement struct {
	globs       []string
	where       Expression
	assignments []Assignment
}

func newUpdateStatement(args []string) (updateStatement, error) {
	globArgs, rest, ok := splitOn(args, "set")
	if !ok {
		return updateStatement{}, fmt.Errorf("missing 'set' keyword")
	}
	assignArgs, whereArgs, _ := splitOn(rest, "where")
	assignArgs = splitCommas(assignArgs)

	if len(globArgs) == 0 {
		return updateStatement{}, fmt.Errorf("no files specified")
	}
	if len(assignArgs) == 0 {
		return updateStatement{}, fmt.Errorf("no fields or assignments specified")
	}

	var assignments []Assignment
	for _, arg := range assignArgs {
		a, err := ParseAssignment(arg)
		if err != nil {
			return updateStatement{}, err
		}
		assignments = append(assignments, a)
	}

	var where Expression
	if len(whereArgs) > 0 {
		var err error
		where, err = ParseExpression(strings.Join(whereArgs, " "))
		if err != nil {
			return updateStatement{}, fmt.Errorf("invalid where clause: %w", err)
		}
	}

	return updateStatement{
		globs:       globArgs,
		where:       where,
		assignments: assignments,
	}, nil
}

func (s updateStatement) run() error {
	paths, err := expandGlobs(s.globs)
	if err != nil {
		return err
	}

	files := readMatchingFiles(paths, s.where)

	var result *SelectResult
	var resultCols []selectCol
	if verbose >= 2 {
		seen := map[string]bool{}
		for _, a := range s.assignments {
			if !seen[a.Field.Name] {
				resultCols = append(resultCols, selectCol{field: a.Field})
				seen[a.Field.Name] = true
			}
		}
		headers := make([]string, len(resultCols))
		for i, c := range resultCols {
			headers[i] = c.field.Name
		}
		result = &SelectResult{headers: headers}
	}

	modified := 0
	for _, f := range files {
		for _, a := range s.assignments {
			if e := f.Apply(a); e != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", f.Path, e)
			}
		}
		if e := f.Write(); e != nil {
			writeErr(f, e)
		} else {
			modified++
			if result != nil {
				result.addRow(f, resultCols)
			}
		}
	}
	if result != nil {
		result.print(os.Stdout)
	} else if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "%d files modified\n", modified)
	}
	return nil
}
