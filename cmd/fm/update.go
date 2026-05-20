package main

import (
	"fmt"
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

func runUpdate(q lib.UpdateQuery, opts options, out, errOut io.Writer) error {
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
		if !q.WhereTruthy(doc.FrontMatter) {
			continue
		}
		if _, err := q.Eval(doc.FrontMatter); err != nil {
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

	return nil
}
