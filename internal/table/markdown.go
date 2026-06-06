package table

import (
	"fmt"
	"io"
	"strings"
)

// Markdown renders a Table as a GitHub-flavoured Markdown pipe table.
type Markdown struct{}

func (Markdown) Render(t Table, w io.Writer) error {
	escape := func(s string) string {
		return strings.ReplaceAll(s, "|", `\|`)
	}

	row := func(cells []string) string {
		escaped := make([]string, len(cells))
		for i, c := range cells {
			escaped[i] = escape(c)
		}
		return "| " + strings.Join(escaped, " | ") + " |"
	}

	if _, err := fmt.Fprintln(w, row(t.Headers)); err != nil {
		return err
	}

	seps := make([]string, len(t.Headers))
	for i := range seps {
		seps[i] = "---"
	}
	if _, err := fmt.Fprintln(w, row(seps)); err != nil {
		return err
	}

	for _, r := range t.Rows {
		if _, err := fmt.Fprintln(w, row(r)); err != nil {
			return err
		}
	}
	return nil
}
