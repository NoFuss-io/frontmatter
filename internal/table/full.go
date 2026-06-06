package table

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Full renders a Table with Unicode box-drawing borders.
type Full struct{}

func (Full) Render(t Table, w io.Writer) error {
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				if n := utf8.RuneCountInString(TruncCell(cell)); n > widths[i] {
					widths[i] = n
				}
			}
		}
	}

	pad := func(s string, w int) string {
		n := utf8.RuneCountInString(s)
		return s + strings.Repeat(" ", w-n)
	}

	hline := func(left, mid, right, horiz rune) string {
		var sb strings.Builder
		sb.WriteRune(left)
		for i, w := range widths {
			sb.WriteString(strings.Repeat(string(horiz), w+2))
			if i < len(widths)-1 {
				sb.WriteRune(mid)
			}
		}
		sb.WriteRune(right)
		return sb.String()
	}

	dataRow := func(cells []string) string {
		var sb strings.Builder
		sb.WriteRune('│')
		for i, w := range widths {
			cell := ""
			if i < len(cells) {
				cell = TruncCell(cells[i])
			}
			sb.WriteString(" " + pad(cell, w) + " ")
			sb.WriteRune('│')
		}
		return sb.String()
	}

	lines := []string{
		hline('┌', '┬', '┐', '─'),
		dataRow(t.Headers),
		hline('├', '┼', '┤', '─'),
	}
	for _, row := range t.Rows {
		lines = append(lines, dataRow(row))
	}
	lines = append(lines, hline('└', '┴', '┘', '─'))

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
