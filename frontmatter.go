// Package frontmatter is the public library API for fm. It re-exports the
// types and functions that downstream Go applications need to parse, run, and
// inspect frontmatter queries against Markdown files.
package frontmatter

import "github.com/nofuss-io/frontmatter/internal"

type (
	FrontMatter = internal.FrontMatter
	Document    = internal.Document
	FilePath    = internal.FilePath

	Program     = internal.Program
	Query       = internal.Query
	SelectQuery = internal.SelectQuery
	UpdateQuery = internal.UpdateQuery
	AlterQuery  = internal.AlterQuery

	ExecOptions = internal.ExecOptions
	Output      = internal.Output
	Table       = internal.Table
	TableRow    = internal.TableRow
)

var (
	ParseProgram = internal.ParseProgram
	ParseQuery   = internal.ParseQuery
	ReadDocument = internal.ReadDocument
	Write        = internal.Write
	ExpandGlobs  = internal.ExpandGlobs
	NewOutput    = internal.NewOutput
	PrintTable   = internal.PrintTable
)
