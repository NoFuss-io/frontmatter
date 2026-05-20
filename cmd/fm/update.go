package main

import (
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

func runUpdate(q lib.UpdateQuery, opts options, _, errOut io.Writer) error {
	return runMutation(q.From, q, opts, errOut)
}
