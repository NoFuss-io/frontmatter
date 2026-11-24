package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/nofuss-io/frontmatter/docs"
)

// helpTopics maps `fm help <topic>` names to embedded Markdown reference pages.
var helpTopics = []struct {
	name, summary string
	body          []byte
}{
	{"syntax", "Query language: statements, fields, types, expressions, globs", docs.ManualMD},
	{"functions", "Built-in functions", docs.FunctionsMD},
}

// writeHelp prints usage (no topic) or the named help topic to out.
func writeHelp(args []string, out io.Writer) error {
	if len(args) == 0 {
		_, err := io.WriteString(out, usage)
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: fm help [topic]")
	}
	names := make([]string, 0, len(helpTopics))
	for _, t := range helpTopics {
		if t.name == args[0] {
			_, err := out.Write(t.body)
			return err
		}
		names = append(names, t.name)
	}
	return fmt.Errorf("unknown help topic %q; available: %s", args[0], strings.Join(names, ", "))
}
