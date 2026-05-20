package internal

func (q UpdateQuery) Eval(fm FrontMatter) (Row, error) {
	if q.Where != nil && !truthy(q.Where.Eval(fm)) {
		return nil, nil
	}
	if err := q.Apply(fm); err != nil {
		return nil, err
	}
	return q.query.evalAlways(fm) // Skip where clause, since values have changed
}

// Apply runs each Assign in order. The first failing assignment halts the
// whole file per Manual.md — the caller skips the write. Where is honored by
// Eval, not here.
func (q UpdateQuery) Apply(fm FrontMatter) error {
	for _, a := range q.Set {
		if err := a.Apply(fm); err != nil {
			return err
		}
	}
	return nil
}
