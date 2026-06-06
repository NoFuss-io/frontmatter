package table

import (
	"encoding/csv"
	"io"
)

// CSV renders a Table as RFC 4180 CSV with a header row.
type CSV struct{}

func (CSV) Render(t Table, w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(t.Headers); err != nil {
		return err
	}
	if err := cw.WriteAll(t.Rows); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}
