package main

import (
	"fmt"
	"io"

	"github.com/nofuss-io/fm/lib"
)

func runSelect(q lib.SelectQuery, _ options, out, errOut io.Writer) error {
	paths, err := lib.ExpandGlobs(q.From)
	if err != nil {
		return err
	}

	var (
		keepPaths []string
		keepRows  []lib.Row
		keepFM    []lib.FrontMatter
	)
	for _, p := range paths {
		f, err := lib.ReadFile(p)
		if err != nil {
			fmt.Fprintf(errOut, "warning: %v\n", err)
			continue
		}
		row, err := q.Eval(&f.FrontMatter)
		if err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", p, err)
			continue
		}
		if row == nil {
			continue
		}
		keepPaths = append(keepPaths, p)
		keepRows = append(keepRows, row)
		keepFM = append(keepFM, f.FrontMatter)
	}

	if len(q.SortBy) > 0 {
		lib.SortRows(keepPaths, keepRows, q.SortBy, keepFM)
	}
	if q.Limit > 0 && len(keepRows) > q.Limit {
		keepPaths = keepPaths[:q.Limit]
		keepRows = keepRows[:q.Limit]
	}

	headers := make([]string, len(q.Fields))
	for i, e := range q.Fields {
		headers[i] = lib.FieldName(e, i)
	}
	lib.PrintTable(out, headers, keepPaths, keepRows)
	return nil
}
