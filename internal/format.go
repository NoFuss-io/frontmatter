package internal

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	tablepkg "github.com/nofuss-io/frontmatter/internal/table"
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
	var tbl tablepkg.Table
	var hiddenCols int

	if t.sel.Star {
		tbl, hiddenCols = t.buildStarTable()
	} else {
		tbl = t.buildTable()
	}

	_ = t.renderer.Render(tbl, w)

	if !t.mutation {
		n := len(t.rows)
		if n == 1 {
			_, _ = fmt.Fprintln(w, "(1 row)")
		} else {
			_, _ = fmt.Fprintf(w, "(%d rows)\n", n)
		}
	}
	if hiddenCols > 0 {
		_, _ = fmt.Fprintf(w, "(%d more column(s) hidden; raise --max-columns to show)\n", hiddenCols)
	}
}

func (t *Table) buildTable() tablepkg.Table {
	headers := make([]string, len(t.sel.Select))
	for i, e := range t.sel.Select {
		headers[i] = FieldName(e, i)
	}
	if !t.noFile {
		headers = append([]string{"filename"}, headers...)
	}
	rows := make([][]string, len(t.rows))
	for i, row := range t.rows {
		if !t.noFile {
			cells := make([]string, len(row.print)+1)
			cells[0] = row.path
			for j, v := range row.print {
				cells[j+1] = FormatValue(v)
			}
			rows[i] = cells
		} else {
			cells := make([]string, len(row.print))
			for j, v := range row.print {
				cells[j] = FormatValue(v)
			}
			rows[i] = cells
		}
	}
	return tablepkg.Table{Headers: headers, Rows: rows}
}

func (t *Table) buildStarTable() (tablepkg.Table, int) {
	seen := make(map[string]struct{})
	for _, r := range t.rows {
		for name := range r.star {
			seen[name] = struct{}{}
		}
	}
	colNames := make([]string, 0, len(seen))
	for name := range seen {
		colNames = append(colNames, name)
	}
	sort.Strings(colNames)

	hidden := 0
	if t.maxColumns > 0 && len(colNames) > t.maxColumns {
		hidden = len(colNames) - t.maxColumns
		colNames = colNames[:t.maxColumns]
	}

	var headers []string
	if !t.noFile {
		headers = append([]string{"filename"}, colNames...)
	} else {
		headers = colNames
	}

	rows := make([][]string, len(t.rows))
	for i, row := range t.rows {
		if !t.noFile {
			cells := make([]string, len(colNames)+1)
			cells[0] = row.path
			for j, name := range colNames {
				if v, ok := row.star[name]; ok {
					cells[j+1] = FormatValue(v)
				}
			}
			rows[i] = cells
		} else {
			cells := make([]string, len(colNames))
			for j, name := range colNames {
				if v, ok := row.star[name]; ok {
					cells[j] = FormatValue(v)
				}
			}
			rows[i] = cells
		}
	}
	return tablepkg.Table{Headers: headers, Rows: rows}, hidden
}

// PrintTable writes a tab-separated table to w: optional filename column plus
// the supplied headers and the materialized values from each TableRow.
// noFile suppresses the filename column (used for FROM-less selects).
func PrintTable(w io.Writer, headers []string, rows []TableRow, noFile bool) {
	var hs []string
	if !noFile {
		hs = append([]string{"filename"}, headers...)
	} else {
		hs = make([]string, len(headers))
		copy(hs, headers)
	}
	tableRows := make([][]string, len(rows))
	for i, row := range rows {
		if !noFile {
			cells := make([]string, len(row.print)+1)
			cells[0] = row.path
			for j, v := range row.print {
				cells[j+1] = FormatValue(v)
			}
			tableRows[i] = cells
		} else {
			cells := make([]string, len(row.print))
			for j, v := range row.print {
				cells[j] = FormatValue(v)
			}
			tableRows[i] = cells
		}
	}
	_ = tablepkg.Simple{}.Render(tablepkg.Table{Headers: hs, Rows: tableRows}, w)
}
