package main

import (
	"fmt"
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

// forEachDoc expands globs, reads each path as a Document, and runs fn.
// Read failures emit a "warning:" line and skip. Errors from fn are reported
// as "path: err" and the loop continues.
func forEachDoc(globs []string, errOut io.Writer, fn func(path string, doc *lib.Document) error) error {
	paths, err := lib.ExpandGlobs(globs)
	if err != nil {
		return err
	}
	for _, p := range paths {
		doc, err := lib.ReadDocument(p)
		if err != nil {
			fmt.Fprintf(errOut, "warning: %v\n", err)
			continue
		}
		if err := fn(p, doc); err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", p, err)
		}
	}
	return nil
}

// mutator is the shared shape of UpdateQuery and AlterQuery for runMutation.
type mutator interface {
	WhereTruthy(fm lib.FrontMatter) bool
	Eval(fm lib.FrontMatter) (lib.Row, error)
}

// runMutation iterates globs, runs the mutator against each document, and
// writes back unless --dry-run is set.
func runMutation(globs []string, q mutator, opts options, errOut io.Writer) error {
	return forEachDoc(globs, errOut, func(p string, doc *lib.Document) error {
		if !q.WhereTruthy(doc.FrontMatter) {
			return nil
		}
		if _, err := q.Eval(doc.FrontMatter); err != nil {
			return err
		}
		if opts.dryRun {
			return nil
		}
		if err := lib.Write(doc, p); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		return nil
	})
}
