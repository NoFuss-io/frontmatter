package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	fm "github.com/nofuss-io/frontmatter"
	"github.com/nofuss-io/frontmatter/internal/table"
)

var (
	Version = "dev"
	Commit  = ""
)

func init() {
	if Version != "dev" && Commit != "" {
		return
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		Version = bi.Main.Version
	}
	if Commit == "" {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				if len(s.Value) > 7 {
					Commit = s.Value[:7]
				} else {
					Commit = s.Value
				}
				break
			}
		}
	}
}

type options struct {
	dryRun             bool
	silent             bool
	verbose            bool
	includeHiddenFiles bool
	maxColumns         int
	format             string
}

var usage = `fm - Markdown frontmatter batch editor
` + Version + `

Usage:
  fm [query] [flags]

Query is read from stdin if omitted. Multiple queries may be separated
by ';' and '--' starts a line comment, so an SQL-style script can be
piped in:

  cat script.sql | fm
  fm < script.sql

Flags:
  -h, --help              Show this message.
  -V, --version           Print version and exit.
  -d, --dry-run           Simulate the operation without editing any files.
  -s, --silent            Suppress all output.
  -v, --verbose           Print affected fields after update or alter.
  -H, --include-hidden    Include hidden files (ignored by default).
      --max-columns N     Column cap for 'select *' output (default 20).
      --format FORMAT     Output format: simple (default), csv, markdown, full.

Subcommands:
  fm completion {bash|zsh|fish}   Print shell-completion script to stdout.
  fm install-skill {claude|codex|copilot|gemini}   Install AI coding-agent skill.
`

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "completion" {
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: fm completion {bash|zsh|fish}")
			os.Exit(2)
		}
		if err := writeCompletion(os.Args[2], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "install-skill" {
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: fm install-skill {claude}")
			os.Exit(2)
		}
		if err := installSkill(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	opts, args, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	okOut := io.Writer(os.Stdout)
	errOut := io.Writer(os.Stderr)

	prog, err := parseProgram(args, os.Stdin)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		os.Exit(2)
	}

	renderer, err := resolveRenderer(opts.format)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		os.Exit(2)
	}
	if ok := prog.Run(fm.ExecOptions{DryRun: opts.dryRun, Silent: opts.silent, Verbose: opts.verbose, MaxColumns: opts.maxColumns, IncludeHidden: opts.includeHiddenFiles, Renderer: renderer}, okOut, errOut); !ok {
		os.Exit(1)
	}
}

func parseFlags(args []string) (options, []string, error) {
	fs := flag.NewFlagSet("fm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts options
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.dryRun, "d", false, "")
	fs.BoolVar(&opts.silent, "silent", false, "")
	fs.BoolVar(&opts.silent, "s", false, "")
	fs.BoolVar(&opts.verbose, "verbose", false, "")
	fs.BoolVar(&opts.verbose, "v", false, "")
	fs.IntVar(&opts.maxColumns, "max-columns", 0, "")
	fs.StringVar(&opts.format, "format", "simple", "")
	fs.BoolVar(&opts.includeHiddenFiles, "include-hidden", false, "")
	fs.BoolVar(&opts.includeHiddenFiles, "hidden", false, "")
	fs.BoolVar(&opts.includeHiddenFiles, "H", false, "")
	help := fs.Bool("help", false, "")
	helpShort := fs.Bool("h", false, "")
	version := fs.Bool("version", false, "")
	versionShort := fs.Bool("V", false, "")

	split := len(args)
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			split = i
			break
		}
	}

	if err := fs.Parse(args[split:]); err != nil {
		return opts, nil, err
	}
	if *help || *helpShort {
		fmt.Print(usage)
		os.Exit(0)
	}
	if *version || *versionShort {
		fmt.Println(Version)
		os.Exit(0)
	}
	return opts, append(args[:split], fs.Args()...), nil
}

func parseProgram(args []string, in io.Reader) (*fm.Program, error) {
	src, err := readProgramString(args, in)
	if err != nil {
		return nil, err
	}
	prog, err := fm.ParseProgram(strings.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(prog.Stmts) == 0 {
		return nil, fmt.Errorf("empty query")
	}
	return &prog, nil
}

func resolveRenderer(format string) (table.Renderer, error) {
	switch format {
	case "", "simple":
		return table.Simple{}, nil
	case "csv":
		return table.CSV{}, nil
	case "markdown", "md":
		return table.Markdown{}, nil
	case "full":
		return table.Full{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q: must be simple, csv, markdown, or full", format)
	}
}

func readProgramString(args []string, in io.Reader) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if len(args) > 1 {
		return "", fmt.Errorf("expected single query argument, got %d", len(args))
	}
	b, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
