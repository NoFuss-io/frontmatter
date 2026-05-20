package main

import (
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

func runSelect(q lib.SelectQuery, _ options, out, errOut io.Writer) error {
	var (
		keepPaths []string
		keepRows  []lib.Row
		keepFM    []lib.FrontMatter
	)
	err := forEachDoc(q.From, errOut, func(p string, doc *lib.Document) error {
		row, err := q.Eval(doc.FrontMatter)
		if err != nil {
			return err
		}
		if row == nil {
			return nil
		}
		keepPaths = append(keepPaths, p)
		keepRows = append(keepRows, row)
		keepFM = append(keepFM, doc.FrontMatter)
		return nil
	})
	if err != nil {
		return err
	}

	if len(q.SortBy) > 0 {
		lib.SortRows(keepPaths, keepRows, q.SortBy, keepFM)
	}
	keepPaths, keepRows = lib.Limit(q.Limit, keepPaths, keepRows)

	if len(q.Fields) == 0 {
		io.WriteString(out, "filename\n")
		for _, p := range keepPaths {
			io.WriteString(out, p+"\n")
		}
	} else {
		headers := make([]string, len(q.Fields))
		for i, e := range q.Fields {
			headers[i] = lib.FieldName(e, i)
		}
		lib.PrintTable(out, headers, keepPaths, keepRows)
	}
	return nil
}
