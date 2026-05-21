package internal

import (
	"slices"
)

func (p Program) Eval(fp FilePath, fm FrontMatter, out *Output) {
	done := make([]bool, len(p.Stmts))
	for i, q := range p.Stmts {
		res, err := q.Eval(fp, fm)
		if err != nil {
			out.error(fp, err)
		}
		if res != nil {
			done[i] = out.append(fp, i, *res)
		}
	}
}

// Eval evaluates the where clause; if truthy, projects Fields into a Row.
// Returns nil Row if where is falsey or evaluates to null.
func (q SelectQuery) Eval(fp FilePath, fm FrontMatter) (*TableRow, error) {
	if q.evalWhere(fp, fm) {
		return nil, nil
	}
	return q.evalSelect(fp, fm)
}

func (q SelectQuery) evalWhere(fp FilePath, fm FrontMatter) bool {
	return slices.Contains(q.From, fp) && (q.Where == nil || truthy(q.Where.Eval(fm)))
}

func (q SelectQuery) evalSelect(fp FilePath, fm FrontMatter) (*TableRow, error) {
	res := q.newResult(fp)
	for i, f := range q.Select {
		res.print[i] = f.Eval(fm)
	}
	for i, f := range q.SortBy {
		res.sort[i] = f.Eval(fm)
	}
	return res, nil
}

// Apply runs each Assign in order, then returns the projected row. The first
// failing assignment halts the whole file — the caller skips the write.
// Where is honored by Eval, not here.
func (q UpdateQuery) Apply(fp FilePath, fm FrontMatter) (*TableRow, error) {
	if q.evalWhere(fp, fm) {
		return nil, nil
	}
	for _, a := range q.Set {
		if err := a.Apply(fm); err != nil {
			return nil, err
		}
	}
	return q.query.evalSelect(fp, fm)
}

// Apply drops or renames fields on fm, then returns the projected row.
// For drop, the row is projected before deletion; for rename, after.
// Where is honored by Eval, not here.
func (q AlterQuery) Apply(fp FilePath, fm FrontMatter) (*TableRow, error) {
	if q.evalWhere(fp, fm) {
		return nil, nil
	}
	var (
		res *TableRow
		err error
	)
	if q.Op == AlterDrop {
		res, err = q.query.evalSelect(fp, fm)
		if err != nil {
			return nil, err
		}
	}
	switch q.Op {
	case AlterDrop:
		for _, f := range q.Drop {
			delete(fm, f.Name)
		}
	case AlterRename:
		for _, p := range q.Rename {
			if v, ok := fm[p.From]; ok {
				fm[p.To] = v
				delete(fm, p.From)
			}
		}
	}
	if q.Op == AlterRename {
		res, err = q.query.evalSelect(fp, fm)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
func (q query) q() query {
	return q
}
