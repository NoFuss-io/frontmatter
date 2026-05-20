package main

import (
	"io"

	lib "github.com/nofuss-io/frontmatter"
)

func runAlter(q lib.AlterQuery, opts options, _, errOut io.Writer) error {
	return runMutation(q.From, q, opts, errOut)
}
