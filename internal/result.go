package internal

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	tablepkg "github.com/nofuss-io/frontmatter/internal/table"
)

// NewOutput allocates one Table per statement, in source order. maxColumns
// caps the column count rendered for `select *` tables; non-star tables are
// unaffected.
func NewOutput(p *Program, err io.Writer, maxColumns int, renderer tablepkg.Renderer) *Output {
	tables := make([]*Table, len(p.Stmts))
	for i, stmt := range p.Stmts {
		tables[i] = &Table{
			sel:        stmt.q(),
			mutation:   stmt.IsMutation(),
			noFile:     len(stmt.Patterns()) == 0 && !stmt.IsMutation(),
			maxColumns: maxColumns,
			renderer:   renderer,
		}
	}
	return &Output{tables: tables, errors: err}
}

// Output accumulates per-statement result tables across a multi-file evaluation.
type Output struct {
	tables []*Table
	errors io.Writer
}

// Append stamps row with path, stores it in the i'th table, and returns true
// when no further input is needed for that statement.
func (o *Output) Append(path FilePath, index int, row *TableRow) (done bool) {
	row.path = path
	return o.tables[index].append(*row)
}

// Done reports whether statement i has collected enough rows to short-circuit
// remaining files (only true for select with limit and no sort).
func (o *Output) Done(index int) bool {
	return o.tables[index].done()
}

// AllDone reports whether every statement has reached its short-circuit.
func (o *Output) AllDone() bool {
	for _, t := range o.tables {
		if !t.done() {
			return false
		}
	}
	return true
}

// Error reports a per-file evaluation error to the configured error sink.
func (o *Output) Error(path FilePath, err error) {
	_, _ = fmt.Fprintf(o.errors, "ERROR: %s: %s\n", path, err)
}

// Finalize sorts and truncates each table according to its source statement.
func (o *Output) Finalize() {
	for _, t := range o.tables {
		t.finalize()
	}
}

// Print writes every table to w in source order. Multi-statement programs
// separate consecutive tables with a blank line. Empty mutation tables
// (produced when verbose is off) are skipped entirely.
func (o *Output) Print(w io.Writer) {
	first := true
	for _, t := range o.tables {
		if t.mutation && len(t.rows) == 0 {
			continue
		}
		if !first {
			_, _ = fmt.Fprintln(w)
		}
		first = false
		t.print(w)
	}
}

// Table holds the accumulated rows for one statement.
type Table struct {
	sel        query
	mutation   bool
	noFile     bool // true for FROM-less selects: filename column is suppressed
	rows       []TableRow
	maxColumns int
	renderer   tablepkg.Renderer
}

func (t *Table) append(row TableRow) (done bool) {
	t.rows = append(t.rows, row)
	return t.done()
}

func (t *Table) done() bool {
	if t.mutation {
		return false
	}
	return len(t.sel.SortBy) == 0 && t.sel.Limit > 0 && len(t.rows) >= t.sel.Limit
}

func (t *Table) finalize() {
	if len(t.sel.SortBy) > 0 {
		terms := t.sel.SortBy
		slices.SortStableFunc(t.rows, func(a, b TableRow) int {
			for i, term := range terms {
				if i >= len(a.sort) || i >= len(b.sort) {
					return 0
				}
				c := compareValues(a.sort[i], b.sort[i])
				if c == 0 {
					continue
				}
				if term.Desc {
					return -c
				}
				return c
			}
			return 0
		})
	}
	if t.sel.Limit > 0 && len(t.rows) > t.sel.Limit {
		t.rows = t.rows[:t.sel.Limit]
	}
}

// TableRow is one projected row: the file it came from and the materialized
// values for the statement's Select and SortBy expressions. star is populated
// instead of print when the source statement was `select *`.
type TableRow struct {
	path  FilePath
	print []Value
	star  map[string]Value
	sort  []Value
}

func (r TableRow) IsZero() bool { return r.path == "" }

func (q query) newResult() *TableRow {
	r := &TableRow{
		sort: make([]Value, len(q.SortBy)),
	}
	if !q.Star {
		r.print = make([]Value, len(q.Select))
	}
	return r
}

// compareValues returns -1, 0, +1. Null sorts after non-null. Numeric values
// compare numerically; everything else falls back to string form.
func compareValues(a, b Value) int {
	if a.Null && b.Null {
		return 0
	}
	if a.Null {
		return 1
	}
	if b.Null {
		return -1
	}
	if isNumeric(a.Kind) && isNumeric(b.Kind) {
		af, _ := Cast(a, TypeNumber)
		bf, _ := Cast(b, TypeNumber)
		if af.Null || bf.Null {
			return strings.Compare(FormatValue(a), FormatValue(b))
		}
		x, y := af.Data.(float64), bf.Data.(float64)
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	}
	if (a.Kind == TypeDate || a.Kind == TypeDatetime) &&
		(b.Kind == TypeDate || b.Kind == TypeDatetime) {
		ta := a.Data.(time.Time)
		tb := b.Data.(time.Time)
		switch {
		case ta.Before(tb):
			return -1
		case ta.After(tb):
			return 1
		}
		return 0
	}
	return strings.Compare(FormatValue(a), FormatValue(b))
}

func isNumeric(k FieldType) bool {
	return k == TypeInt || k == TypeNumber || k == TypeBool
}
