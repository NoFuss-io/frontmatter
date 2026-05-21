package main

import (
	"io"

	fm "github.com/nofuss-io/frontmatter"
)

func runSelect(q fm.SelectQuery, _ options, out, errOut io.Writer) error {
	var (
		keepPaths []string
		keepRows  []fm.Row
		keepFM    []fm.FrontMatter
	)
	err := forEachDoc(q.From, errOut, func(p string, doc *fm.Document) error {
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
		fm.SortRows(keepPaths, keepRows, q.SortBy, keepFM)
	}
	keepPaths, keepRows = fm.Limit(q.Limit, keepPaths, keepRows)

	if len(q.Select) == 0 {
		io.WriteString(out, "filename\n")
		for _, p := range keepPaths {
			io.WriteString(out, p+"\n")
		}
	} else {
		headers := make([]string, len(q.Select))
		for i, e := range q.Select {
			headers[i] = fm.FieldName(e, i)
		}
		fm.PrintTable(out, headers, keepPaths, keepRows)
	}
	return nil
}
