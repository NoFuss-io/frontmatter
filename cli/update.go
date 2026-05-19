package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/backlin/frontmatter/lib"
)

func runUpdate(q lib.UpdateQuery, opts options, out, errOut io.Writer) error {
	paths, err := lib.ExpandGlobs(q.From)
	if err != nil {
		return err
	}

	var (
		touched   []string
		touchedFM []lib.FrontMatter
	)
	for _, p := range paths {
		f, err := lib.ReadFile(p)
		if err != nil {
			fmt.Fprintf(errOut, "warning: %v\n", err)
			continue
		}
		if q.Where != nil && !truthyExpr(q.Where, &f.FrontMatter) {
			continue
		}
		if err := applyAssignments(q.Set, &f.FrontMatter); err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", p, err)
			continue
		}
		if !opts.dryRun {
			if err := f.Write(); err != nil {
				fmt.Fprintf(errOut, "%s: write: %v\n", p, err)
				continue
			}
		}
		touched = append(touched, p)
		touchedFM = append(touchedFM, f.FrontMatter)
	}

	if opts.verbose {
		printAffected(out, affectedFields(q.Set), touched, touchedFM)
	}
	return nil
}

// applyAssignments applies each Assign in order. The first failing assignment
// halts the whole file per Manual.md — the caller skips the write.
func applyAssignments(assigns []lib.Assign, fm *lib.FrontMatter) error {
	for _, a := range assigns {
		if err := a.Apply(fm); err != nil {
			return err
		}
	}
	return nil
}

// truthyExpr mirrors the Manual.md rule: a null or cast-failed where is falsey.
func truthyExpr(e lib.Expr, fm *lib.FrontMatter) bool {
	v := e.Eval(fm)
	if v.Null {
		return false
	}
	if v.Kind == lib.TypeBool {
		return v.Data.(bool)
	}
	c, err := lib.Cast(v, lib.TypeBool)
	if err != nil || c.Null {
		return false
	}
	return c.Data.(bool)
}

func affectedFields(assigns []lib.Assign) []lib.Field {
	seen := map[string]bool{}
	var out []lib.Field
	for _, a := range assigns {
		if seen[a.Field.Name] {
			continue
		}
		seen[a.Field.Name] = true
		out = append(out, a.Field)
	}
	return out
}

// printAffected renders a select-style table of the touched files restricted
// to the supplied fields.
func printAffected(out io.Writer, fields []lib.Field, paths []string, fms []lib.FrontMatter) {
	headers := make([]string, len(fields))
	exprs := make([]lib.Expr, len(fields))
	for i, f := range fields {
		headers[i] = f.Name
		exprs[i] = lib.FieldExpr{Field: f}
	}
	rows := make([]lib.Row, len(paths))
	for i := range paths {
		row := make(lib.Row, len(exprs))
		for j, e := range exprs {
			row[j] = e.Eval(&fms[i])
		}
		rows[i] = row
	}
	sortByPath(paths, rows)
	lib.PrintTable(out, headers, paths, rows)
}

func sortByPath(paths []string, rows []lib.Row) {
	type entry struct {
		path string
		row  lib.Row
	}
	es := make([]entry, len(paths))
	for i := range paths {
		es[i] = entry{paths[i], rows[i]}
	}
	sort.SliceStable(es, func(i, j int) bool { return es[i].path < es[j].path })
	for i := range es {
		paths[i] = es[i].path
		rows[i] = es[i].row
	}
}
