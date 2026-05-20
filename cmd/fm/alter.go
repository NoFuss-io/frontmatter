package main

import (
	"fmt"
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

func runAlter(q lib.AlterQuery, opts options, out, errOut io.Writer) error {
	paths, err := lib.ExpandGlobs(q.From)
	if err != nil {
		return err
	}

	var (
		touched   []string
		touchedFM []lib.FrontMatter
	)
	for _, p := range paths {
		doc, err := lib.ReadDocument(p)
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
			if err := lib.Write(doc, p); err != nil {
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

func alterAffectedFields(q lib.AlterQuery) []lib.Field {
	switch q.Op {
	case lib.AlterDrop:
		return append([]lib.Field(nil), q.Drop...)
	case lib.AlterRename:
		out := make([]lib.Field, len(q.Rename))
		for i, r := range q.Rename {
			out[i] = lib.Field{Name: r.To}
		}
		return out
	}
	return nil
}
