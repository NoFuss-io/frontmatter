package main

import (
	"fmt"
	"io"

	fm "github.com/nofuss-io/frontmatter"
)

func runAlter(q fm.AlterQuery, opts options, out, errOut io.Writer) error {
	paths, err := fm.ExpandGlobs(q.From)
	if err != nil {
		return err
	}

	var (
		touched   []string
		touchedFM []fm.FrontMatter
	)
	for _, p := range paths {
		doc, err := fm.ReadDocument(p)
		if err != nil {
			fmt.Fprintf(errOut, "warning: %v\n", err)
			continue
		}
		if q.Where != nil && !truthyExpr(q.Where, &doc.FrontMatter) {
			continue
		}
		if err := q.Apply(&doc.FrontMatter); err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", p, err)
			continue
		}
		if !opts.dryRun {
			if err := fm.Write(doc, p); err != nil {
				fmt.Fprintf(errOut, "%s: write: %v\n", p, err)
				continue
			}
		}
		touched = append(touched, p)
		touchedFM = append(touchedFM, doc.FrontMatter)
	}

	if opts.verbose {
		printAffected(out, alterAffectedFields(q), touched, touchedFM)
	}
	return nil
}

func alterAffectedFields(q fm.AlterQuery) []fm.Field {
	switch q.Op {
	case fm.AlterDrop:
		return append([]fm.Field(nil), q.Drop...)
	case fm.AlterRename:
		out := make([]fm.Field, len(q.Rename))
		for i, r := range q.Rename {
			out[i] = fm.Field{Name: r.To}
		}
		return out
	}
	return nil
}
