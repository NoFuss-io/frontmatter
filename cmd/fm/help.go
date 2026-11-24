package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/nofuss-io/frontmatter/docs"
)

// helpTopics maps `fm help <topic>` names to embedded Markdown reference pages.
// The usage text lists them via helpTopicList.
var helpTopics = []struct {
	name, summary string
	body          []byte
}{
	{"syntax", "Query language reference", docs.ManualMD},
	{"functions", "Built-in functions", docs.FunctionsMD},
}

// helpTopicList renders the topic lines shown under `fm help [topic]` in usage.
func helpTopicList() string {
	var b strings.Builder
	for _, t := range helpTopics {
		fmt.Fprintf(&b, "%36s%-11s%s\n", "", t.name, t.summary)
	}
	return b.String()
}

// installSkillRemoved is printed by the removed `fm install-skill` subcommand.
const installSkillRemoved = `fm install-skill has been removed. The fm skill is now distributed as an
agent plugin; see https://github.com/NoFuss-io/frontmatter#ai-agent-plugin

Skills installed by earlier versions are outdated and should be deleted:
  ~/.claude/skills/fm  ~/.codex/skills/fm  ~/.copilot/skills/fm  ~/.gemini/skills/fm
`

// writeHelp prints usage (no topic) or the named help topic to out.
func writeHelp(args []string, out io.Writer) error {
	if len(args) == 0 {
		_, err := io.WriteString(out, usageText())
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
