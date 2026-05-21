package internal

// Eval projects Select fields into a TableRow, honoring the where clause.
// Returns (nil, nil) when where is falsey or null. Pure read; never mutates fm.
func (q query) Eval(fm FrontMatter) (*TableRow, error) {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil, nil
	}
	return q.evalSelect(fm), nil
}

func (q query) IsMutation() bool { return false }

func (q query) Globs() []string { return q.From }

func (q query) q() query { return q }

func (q query) evalSelect(fm FrontMatter) *TableRow {
	res := q.newResult()
	for i, f := range q.Select {
		res.print[i] = f.Eval(fm)
	}
	for i, f := range q.SortBy {
		res.sort[i] = f.Eval(fm)
	}
	return res
}

// Eval applies all assignments in order, then projects the affected fields.
// A failed assignment returns (nil, err); fm may be partially mutated.
// Caller is responsible for discarding the file on error.
func (q UpdateQuery) Eval(fm FrontMatter) (*TableRow, error) {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil, nil
	}
	for _, a := range q.Set {
		if err := a.Apply(fm); err != nil {
			return nil, err
		}
	}
	return q.query.evalSelect(fm), nil
}

func (q UpdateQuery) IsMutation() bool { return true }

// Eval drops or renames fields, then projects the affected fields.
// For drop, projection happens before deletion so dropped values are visible.
// For rename, projection happens after rename so headers match new names.
func (q AlterQuery) Eval(fm FrontMatter) (*TableRow, error) {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil, nil
	}
	switch q.Op {
	case AlterDrop:
		res := q.query.evalSelect(fm)
		for _, f := range q.Drop {
			delete(fm, f.Name)
		}
		return res, nil
	case AlterRename:
		for _, p := range q.Rename {
			if v, ok := fm[p.From]; ok {
				fm[p.To] = v
				delete(fm, p.From)
			}
		}
		return q.query.evalSelect(fm), nil
	}
	return nil, nil
}

func (q AlterQuery) IsMutation() bool { return true }
