package internal

import "fmt"

// passesWhere reports whether fm satisfies q.Where. A nil clause always passes.
// An eval error propagates so the caller can halt and discard the file.
func (q query) passesWhere(fm FrontMatter) (bool, error) {
	if q.Where == nil {
		return true, nil
	}
	w, err := q.Where.Eval(fm)
	if err != nil {
		return false, fmt.Errorf("could not evaluate where clause: %w", err)
	}
	return truthy(w), nil
}

// Eval projects Select fields into a TableRow, honoring the where clause.
// Returns (nil, nil) when where is falsey or null. Pure read; never mutates fm.
func (q query) Eval(fm FrontMatter) (*TableRow, error) {
	pass, err := q.passesWhere(fm)
	if err != nil {
		return nil, err
	}
	if !pass {
		return nil, nil
	}
	return q.evalSelect(fm)
}

func (q query) IsMutation() bool { return false }

func (q query) Patterns() []string { return q.From }

func (q query) q() query { return q }

func (q query) evalSelect(fm FrontMatter) (*TableRow, error) {
	res := q.newResult()
	if q.Star {
		res.star = make(map[string]Value, len(fm))
		for name, raw := range fm {
			res.star[name] = valueFromAny(raw)
		}
	} else {
		for i, f := range q.Select {
			v, err := f.Eval(fm)
			if err != nil {
				return nil, fmt.Errorf("could not evaluate select expression %d: %w", i+1, err)
			}
			res.print[i] = v
		}
	}
	for i, f := range q.SortBy {
		v, err := f.Eval(fm)
		if err != nil {
			return nil, fmt.Errorf("could not evaluate sort expression %d: %w", i+1, err)
		}
		res.sort[i] = v
	}
	return res, nil
}

// Eval applies all assignments in order, then projects the affected fields.
// A failed assignment returns (nil, err); fm may be partially mutated.
// Caller is responsible for discarding the file on error.
func (q UpdateQuery) Eval(fm FrontMatter) (*TableRow, error) {
	pass, err := q.passesWhere(fm)
	if err != nil {
		return nil, err
	}
	if !pass {
		return nil, nil
	}
	for _, a := range q.Set {
		if err := a.Apply(fm); err != nil {
			return nil, err
		}
	}
	return q.evalSelect(fm)
}

func (q UpdateQuery) IsMutation() bool { return true }

// Eval drops or renames fields, then projects the affected fields.
// For drop, projection happens before deletion so dropped values are visible.
// For rename, projection happens after rename so headers match new names.
func (q AlterQuery) Eval(fm FrontMatter) (*TableRow, error) {
	pass, err := q.passesWhere(fm)
	if err != nil {
		return nil, err
	}
	if !pass {
		return nil, nil
	}
	switch q.Op {
	case AlterDrop:
		res, err := q.evalSelect(fm)
		if err != nil {
			return nil, err
		}
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
		return q.evalSelect(fm)
	}
	return nil, nil
}

func (q AlterQuery) IsMutation() bool { return true }
