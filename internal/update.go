package internal

import (
	"io"
	"sort"
)

func (u UpdateQuery) Eval(fm FrontMatter) (Row, error) {
	if err := u.Apply(fm); err != nil {
		return nil, err
	}
	return u.query.Eval(fm)
}

// Apply evaluates the where clause; if truthy, applies each Assign to fm.
func (q UpdateQuery) Apply(fm FrontMatter) error {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil
	}
	for _, a := range q.Set {
		if err := a.Apply(fm); err != nil {
			return err
		}
	}
	return nil
}

// UNUSED?

// applyAssignments applies each Assign in order. The first failing assignment
// halts the whole file per Manual.md — the caller skips the write.
func applyAssignments(assigns []Assign, fm FrontMatter) error {
	for _, a := range assigns {
		if err := a.Apply(fm); err != nil {
			return err
		}
	}
	return nil
}

func affectedFields(assigns []Assign) []Field {
	seen := map[string]bool{}
	var out []Field
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
func printAffected(out io.Writer, fields []Field, paths []string, fms []FrontMatter) {
	headers := make([]string, len(fields))
	exprs := make([]Expr, len(fields))
	for i, f := range fields {
		headers[i] = f.Name
		exprs[i] = FieldExpr{Field: f}
	}
	rows := make([]Row, len(paths))
	for i := range paths {
		row := make(Row, len(exprs))
		for j, e := range exprs {
			row[j] = e.Eval(fms[i])
		}
		rows[i] = row
	}
	sortByPath(paths, rows)
	PrintTable(out, headers, paths, rows)
}

func sortByPath(paths []string, rows []Row) {
	type entry struct {
		path string
		row  Row
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
