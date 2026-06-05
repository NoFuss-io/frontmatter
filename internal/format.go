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

func (t *Table) print(w io.Writer) {
	if t.sel.Star {
		printStarTable(w, t.rows, t.maxColumns)
		return
	}
	headers := make([]string, len(t.sel.Select))
	for i, e := range t.sel.Select {
		headers[i] = FieldName(e, i)
	}
	PrintTable(w, headers, t.rows)
}

// printStarTable renders rows produced by `select *`. The header set is the
// alphabetical union of field names seen across all rows, capped at
// maxColumns; any extra columns are dropped and a trailing note reports the
// hidden count. maxColumns <= 0 disables the cap.
func printStarTable(w io.Writer, rows []TableRow, maxColumns int) {
	seen := make(map[string]struct{})
	for _, r := range rows {
		for name := range r.star {
			seen[name] = struct{}{}
		}
	}
	headers := make([]string, 0, len(seen))
	for name := range seen {
		headers = append(headers, name)
	}
	sort.Strings(headers)

	hidden := 0
	if maxColumns > 0 && len(headers) > maxColumns {
		hidden = len(headers) - maxColumns
		headers = headers[:maxColumns]
	}

	showFile := len(rows) == 0
	for _, row := range rows {
		if row.path != "" {
			showFile = true
			break
		}
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var hs []string
	if showFile {
		hs = append([]string{"filename"}, headers...)
	} else {
		hs = make([]string, len(headers))
		copy(hs, headers)
	}
	seps := make([]string, len(hs))
	for i, h := range hs {
		seps[i] = strings.Repeat("-", len(h))
	}
	_, _ = fmt.Fprintln(tw, strings.Join(hs, "\t"))
	_, _ = fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for _, row := range rows {
		var cells []string
		if showFile {
			cells = make([]string, len(headers)+1)
			cells[0] = truncCell(row.path)
			for j, name := range headers {
				if v, ok := row.star[name]; ok {
					cells[j+1] = truncCell(FormatValue(v))
				}
			}
		} else {
			cells = make([]string, len(headers))
			for j, name := range headers {
				if v, ok := row.star[name]; ok {
					cells[j] = truncCell(FormatValue(v))
				}
			}
		}
		_, _ = fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	_ = tw.Flush()
	if hidden > 0 {
		_, _ = fmt.Fprintf(w, "(%d more column(s) hidden; raise --max-columns to show)\n", hidden)
	}
}

// PrintTable writes a tab-separated table to w: filename column plus the
// supplied headers and the materialized values from each TableRow.
// The filename column is omitted when all rows are from a FROM-less select
// (all paths empty). Empty result sets always include the filename column.
func PrintTable(w io.Writer, headers []string, rows []TableRow) {
	showFile := len(rows) == 0
	for _, row := range rows {
		if row.path != "" {
			showFile = true
			break
		}
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var hs []string
	if showFile {
		hs = append([]string{"filename"}, headers...)
	} else {
		hs = make([]string, len(headers))
		copy(hs, headers)
	}
	seps := make([]string, len(hs))
	for i, h := range hs {
		seps[i] = strings.Repeat("-", len(h))
	}
	_, _ = fmt.Fprintln(tw, strings.Join(hs, "\t"))
	_, _ = fmt.Fprintln(tw, strings.Join(seps, "\t"))
	for _, row := range rows {
		var cells []string
		if showFile {
			cells = make([]string, len(row.print)+1)
			cells[0] = truncCell(row.path)
			for j, v := range row.print {
				cells[j+1] = truncCell(FormatValue(v))
			}
		} else {
			cells = make([]string, len(row.print))
			for j, v := range row.print {
				cells[j] = truncCell(FormatValue(v))
			}
		}
		_, _ = fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	_ = tw.Flush()
}

// maxCellWidth caps the rune length of any printed table cell. Longer strings
// are truncated with a trailing ellipsis so columns stay readable.
const maxCellWidth = 30

func truncCell(s string) string {
	runes := []rune(s)
	if len(runes) <= maxCellWidth {
		return s
	}
	return string(runes[:maxCellWidth-1]) + "…"
}
