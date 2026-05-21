package internal

import (
	"fmt"
	"io"
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

func (t *Table) Print(w io.Writer, headers []string) {
	PrintTable(w, t.Headers, t.Paths, t.Rows)
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
