package main

import (
	"fmt"
	"os"
	"path/filepath"
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

  field:type      Cast field to type; creates it as null if absent. Skipped when type is any.
  field=value     Set field to value. Use null to clear the field.
  field+=value    Numbers: add. Strings: append. Lists: append if not already present.
  field-=value    Numbers: subtract. Lists: remove if present.`,
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
	var paths []string
	for _, p := range s.globs {
		if strings.ContainsAny(p, "*?[") {
			expanded, err := filepath.Glob(p)
			if err != nil {
				return err
			}
			paths = append(paths, expanded...)
		} else if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		} else {
			return fmt.Errorf("no such file or pattern: %s", p)
		}
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
		for _, a := range s.assignments {
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
}
