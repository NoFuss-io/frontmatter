package internal

import (
	"fmt"
	"io"
	"slices"

	tablepkg "github.com/nofuss-io/frontmatter/internal/table"
	"github.com/nofuss-io/frontmatter/store"
	markdownstore "github.com/nofuss-io/frontmatter/store/markdown"
)

// ExecOptions controls how a Program is executed.
type ExecOptions struct {
	// DryRun runs all statements in memory but does not write mutated documents back to disk.
	DryRun bool
	// Silent suppresses all normal output (errors are still written to errOut).
	Silent bool
	// Verbose prints the affected fields after update or alter statements.
	// Without this flag, mutation statements produce no output.
	Verbose bool
	// MaxColumns caps the number of frontmatter columns rendered for `select *`
	// output. Zero means use DefaultMaxColumns. Excess columns are dropped from
	// the printed table; row count is unaffected.
	MaxColumns int
	// IncludeHidden includes files whose basename begins with '.' in glob
	// expansion. Default skips them.
	IncludeHidden bool
	// Renderer controls how result tables are formatted. Nil defaults to Simple.
	Renderer tablepkg.Renderer
	// Store selects the data-source backend. Nil defaults to
	// store/markdown.New() — preserving the current behaviour.
	Store store.Store
}

// DefaultMaxColumns is the column cap applied to `select *` output when
// ExecOptions does not set an explicit value.
const DefaultMaxColumns = 20

// Run executes the program. For each file in the union of all statement globs,
// the document is read once, every applicable statement runs against the
// in-memory frontmatter, and the file is written back if any mutation
// succeeded. A failing statement halts further work on that file and cancels
// its write; the loop proceeds to the next file. Output for completed
// statements is preserved. Tables are written to okOut in source order;
// per-file errors are reported to errOut. The returned bool is false if any
// file failed to read, write, or evaluate.
func (p Program) Run(opts ExecOptions, okOut, errOut io.Writer) (ok bool) {
	ok = true

	maxColumns := opts.MaxColumns
	if maxColumns <= 0 {
		maxColumns = DefaultMaxColumns
	}

	if opts.Silent {
		okOut = io.Discard
	}

	renderer := opts.Renderer
	if renderer == nil {
		renderer = tablepkg.Simple{}
	}
	out := NewOutput(&p, errOut, maxColumns, renderer)

	s := opts.Store
	if s == nil {
		s = markdownstore.New()
	}

	stmtPaths, allPaths, err := expandPlan(p, s, opts.IncludeHidden)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return false
	}

	for _, key := range allPaths {
		if out.AllDone() {
			break
		}

		fm, err := s.Read(key)
		if err != nil {
			ok = false
			out.Error(s.Label(key), fmt.Errorf("could not read: %w", err))
			continue
		}

		mutated := false
		halted := false
		for i, stmt := range p.Stmts {
			if !slices.Contains(stmtPaths[i], key) {
				continue
			}
			if out.Done(i) {
				continue
			}
			row, evalErr := stmt.Eval(FrontMatter(fm))
			if evalErr != nil {
				ok = false
				out.Error(s.Label(key), evalErr)
				halted = true
				break
			}
			if row == nil {
				continue
			}
			if stmt.IsMutation() {
				mutated = true
				if !opts.Verbose {
					continue
				}
			}
			out.Append(s.Label(key), i, row)
		}

		if mutated && !halted && !opts.DryRun {
			if err := s.Write(key, fm); err != nil {
				ok = false
				out.Error(s.Label(key), fmt.Errorf("could not write: %w", err))
			}
		}
	}

	// Handle FROM-less select queries: evaluate once against empty frontmatter.
	for i, stmt := range p.Stmts {
		if len(stmtPaths[i]) == 0 && !stmt.IsMutation() {
			row, evalErr := stmt.Eval(FrontMatter{})
			if evalErr != nil {
				ok = false
				_, _ = fmt.Fprintln(errOut, evalErr)
				continue
			}
			if row != nil {
				out.Append("", i, row)
			}
		}
	}

	out.Finalize()
	out.Print(okOut)
	return ok
}

// expandPlan resolves each statement's patterns once via the store and returns
// the per-statement key lists plus the deduplicated union iterated by the outer
// file loop.
func expandPlan(p Program, s store.Store, includeHidden bool) ([][]string, []string, error) {
	stmtPaths := make([][]string, len(p.Stmts))
	var all []string
	seen := make(map[string]bool)
	for i, stmt := range p.Stmts {
		keys, err := s.Enumerate(stmt.Patterns(), store.EnumOptions{IncludeHidden: includeHidden})
		if err != nil {
			return nil, nil, fmt.Errorf("statement %d: %w", i+1, err)
		}
		stmtPaths[i] = keys
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				all = append(all, k)
			}
		}
	}
	return stmtPaths, all, nil
}
