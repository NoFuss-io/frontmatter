package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	fm "github.com/nofuss-io/frontmatter"
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
}

var usage = `fm -- Markdown frontmatter batch editor
` + Version + ` (` + Commit + `)

Usage:
  fm [query] [flags]

Query is read from stdin if omitted. Multiple queries may be separated
by ';' and '--' starts a line comment, so an SQL-style script can be
piped in:

  fm < script.sql

Flags:
  -h, --help              Show this message.
  -d, --dry-run           Simulate the operation without editing any files.
  -H, --hidden            Include hidden files (ignored by default).
      --max-columns N     Column cap for 'select *' output (default 20).

Subcommands:
  fm completion {bash|zsh|fish}   Print shell-completion script to stdout.
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
		fmt.Fprintln(errOut, err)
		os.Exit(2)
	}

	if ok := prog.Run(fm.ExecOptions{DryRun: opts.dryRun, MaxColumns: opts.maxColumns, IncludeHidden: opts.includeHiddenFiles}, okOut, errOut); !ok {
		os.Exit(1)
	}
}

func parseFlags(args []string) (options, []string, error) {
	fs := flag.NewFlagSet("fm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts options
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.silent, "silent", false, "")
	fs.BoolVar(&opts.verbose, "v", false, "")
	fs.IntVar(&opts.maxColumns, "max-columns", 0, "")
	fs.BoolVar(&opts.includeHiddenFiles, "hidden", false, "")
	fs.BoolVar(&opts.includeHiddenFiles, "H", false, "")
	help := fs.Bool("help", false, "")
	helpShort := fs.Bool("h", false, "")

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
	return opts, args[:split], nil
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
