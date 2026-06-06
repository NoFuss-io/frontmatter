package table

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const MaxCellWidth = 30

// Simple renders a Table as tab-aligned plain text with a dashed separator row.
type Simple struct{}

func (Simple) Render(t Table, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	seps := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		seps[i] = strings.Repeat("-", len(h))
	}
	if _, err := fmt.Fprintln(tw, strings.Join(t.Headers, "\t")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, strings.Join(seps, "\t")); err != nil {
		return err
	}
	for _, row := range t.Rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = TruncCell(cell)
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// TruncCell caps s to MaxCellWidth runes, appending an ellipsis if truncated.
func TruncCell(s string) string {
	runes := []rune(s)
	if len(runes) <= MaxCellWidth {
		return s
	}
	return string(runes[:MaxCellWidth-1]) + "…"
}
