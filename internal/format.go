package internal

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// FieldName returns a sensible column header for an expression. Field references
// use the field name; anything else falls back to a positional placeholder.
func FieldName(e Expr, idx int) string {
	if f, ok := e.(FieldExpr); ok {
		return f.Field.Name
	}
	return fmt.Sprintf("expr%d", idx+1)
}

// FormatValue renders a Value as a plain string for table output.
func FormatValue(v Value) string {
	if v.Null {
		return ""
	}
	switch v.Kind {
	case TypeString, TypeLink, TypeMdLink:
		return v.Data.(string)
	case TypeInt:
		return fmt.Sprintf("%d", v.Data.(int64))
	case TypeNumber:
		return fmt.Sprintf("%g", v.Data.(float64))
	case TypeBool:
		if v.Data.(bool) {
			return "true"
		}
		return "false"
	case TypeDate:
		return v.Data.(time.Time).Format("2006-01-02")
	case TypeDatetime:
		return v.Data.(time.Time).Format("2006-01-02T15:04:05")
	case TypeList:
		els := v.Data.([]Value)
		parts := make([]string, len(els))
		for i, e := range els {
			parts[i] = FormatValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case TypeAny:
		return fmt.Sprintf("%v", v.Data)
	}
	return ""
}

// PrintTable writes a tab-separated table to w: filename column plus the
// supplied headers/rows.
func PrintTable(w io.Writer, headers []string, paths []string, rows []Row) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	hs := append([]string{"filename"}, headers...)
	seps := make([]string, len(hs))
	for i, h := range hs {
		seps[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(hs, "\t"))
	fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for i, row := range rows {
		cells := make([]string, len(row)+1)
		cells[0] = paths[i]
		for j, v := range row {
			cells[j+1] = FormatValue(v)
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	tw.Flush()
}

// SortRows reorders paths/rows in place according to the SortTerms.
// Null values sort last. Default direction asc unless term.Desc.
func SortRows(paths []string, rows []Row, terms []SortTerm, frontmatter []FrontMatter) {
	type item struct {
		path string
		row  Row
		fm   FrontMatter
	}
	items := make([]item, len(paths))
	for i := range paths {
		items[i] = item{paths[i], rows[i], frontmatter[i]}
	}
	sort.SliceStable(items, func(i, j int) bool {
		for _, t := range terms {
			vi := t.Eval(&items[i].fm)
			vj := t.Eval(&items[j].fm)
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
		frontmatter[i] = items[i].fm
	}
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
