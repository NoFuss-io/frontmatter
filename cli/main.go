package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type Semver struct{ Major, Minor, Patch int }

func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

var VERSION = Semver{0, 1, 0}

var verbose int

func main() {
	root := &cobra.Command{
		Use:   "fm",
		Short: fmt.Sprintf("Markdown frontmatter batch editor (%s)", VERSION),
		Long: `fm is a command-line tool for batch-querying and editing YAML frontmatter in
Markdown files. It is designed for knowledge management workflows such as Obsidian
vaults, where structured metadata lives in a --- YAML block at the top of each .md file.

Commands follow a SQL-inspired syntax:

  fm select <fields> from <files> [where <expr>] [sort by <field> [desc]] [limit <n>]
  fm update <files> set <assignments> [where <expr>]
  fm alter  <files> drop <fields> [where <expr>]`,
		SilenceUsage: true,
	}
	root.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Verbosity: -v prints modified count, -vv prints each processed file and total")
	root.AddCommand(selectCmd(), updateCmd(), alterCmd())
	root.AddCommand(genManCmd(root), installManCmd(root))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func writeErr(f *File, err error) {
	fmt.Fprintf(os.Stderr, "error writing %s: %v\n", f.Path, err)
}
