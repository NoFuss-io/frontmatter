package table

import "io"

type Table struct {
	Headers []string
	Rows    [][]string
}

type Renderer interface {
	Render(table Table, w io.Writer) error
}
