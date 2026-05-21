package internal

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

func NewOutput(p *Program, err io.Writer) *Output {
	tables := make([]Table, len(p.Stmts))
	for i, stmt := range p.Stmts {
		tables[i] = Table{sel: stmt.q()}
	}
	return &Output{
		tables: tables,
		errors: err,
	}
}

type Output struct {
	tables []Table
	errors io.Writer
}

func (o *Output) append(path FilePath, index int, row TableRow) (done bool) {
	o.tables[index].append(path, row)
	return true
}

func (o *Output) error(path FilePath, err error) {
	fmt.Fprintf(o.errors, "ERROR: %s: %s", path, err)
}

type Table struct {
	sel   SelectQuery
	rows  Rows
	limit int
}

type Rows []TableRow

func (t Table) append(path FilePath, row TableRow) (done bool) {
	t.rows = append(t.rows, row)
	return len(t.rows) >= t.limit
}

func (t Rows) Sort() {
	slices.SortStableFunc(t, func(a, b TableRow) int {
		for i := range a.sort {
			if c := compareValues(a.sort[i], b.sort[i]); c != 0 {
				return c
			}
		}
		return 0
	})
}

type TableRow struct {
	path  FilePath
	print []Value
	sort  []Value
}

func (r TableRow) IsZero() bool {
	return r.path == ""
}

func (q query) newResult(path FilePath) *TableRow {
	return &TableRow{
		path:  path,
		print: make([]Value, 0, len(q.Select)),
		sort:  make([]Value, 0, len(q.SortBy)),
	}
}

// Row is the projected output of SelectQuery.Eval for a single document.
// The slice index matches the order of SelectQuery.Fields.
type Row []Value

// SortRows reorders paths/rows in place according to terms. Null values sort
// last. Default direction asc unless term.Desc.
func SortRows(paths []FilePath, rows []Row, terms []SortTerm, fms []FrontMatter) {
	type item struct {
		path string
		row  Row
		fm   FrontMatter
	}
	items := make([]item, len(paths))
	for i := range paths {
		items[i] = item{paths[i], rows[i], fms[i]}
	}
	sort.SliceStable(items, func(i, j int) bool {
		for _, t := range terms {
			vi := t.Eval(items[i].fm)
			vj := t.Eval(items[j].fm)
			c := compareValues(vi, vj)
			if c == 0 {
				continue
			}
			if t.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	for i := range items {
		paths[i] = items[i].path
		rows[i] = items[i].row
		fms[i] = items[i].fm
	}
}

// Limit truncates paths and rows to n. n <= 0 means no limit.
func Limit(n int, paths []string, rows []Row) ([]string, []Row) {
	if n > 0 && len(rows) > n {
		return paths[:n], rows[:n]
	}
	return paths, rows
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
