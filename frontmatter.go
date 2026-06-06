// Package frontmatter is the public library API for fm. It re-exports the
// types and functions that downstream Go applications need to parse, run, and
// inspect frontmatter queries against Markdown files.
package frontmatter

import (
	"github.com/nofuss-io/frontmatter/internal"
	"github.com/nofuss-io/frontmatter/store"
	"github.com/nofuss-io/frontmatter/store/markdown"
)

type (
	FrontMatter = internal.FrontMatter
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

	// Store types — re-exported so library consumers import only this package.
	Store       = store.Store
	Format      = store.Format
	FileStore   = store.FileStore
	EnumOptions = store.EnumOptions
)

var (
	ParseProgram = internal.ParseProgram
	ParseQuery   = internal.ParseQuery
	NewOutput    = internal.NewOutput
	PrintTable   = internal.PrintTable

	// NewMarkdownStore constructs the default markdown store. Library consumers
	// can pass the result as ExecOptions.Store or use it directly.
	NewMarkdownStore = markdown.New
)
