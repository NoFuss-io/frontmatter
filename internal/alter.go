package internal

// Eval applies the alter (drop/rename) to fm via Apply.
func (q AlterQuery) Eval(fm FrontMatter) (Row, error) {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil, nil
	}
	var row Row
	var err error
	if q.Op == AlterDrop {
		row, err = q.query.evalAlways(fm)
		if err != nil {
			return nil, err
		}
	}
	if err := q.Apply(fm); err != nil {
		return nil, err
	}
	if q.Op == AlterRename {
		row, err = q.query.evalAlways(fm)
		if err != nil {
			return nil, err
		}
	}
	return row, nil
}

// Apply evaluates the where clause; if truthy, drops or renames fields on fm.
func (q AlterQuery) Apply(fm FrontMatter) error {
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
	return nil
}
