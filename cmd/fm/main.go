package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nofuss-io/frontmatter/internal"
	fm "github.com/nofuss-io/frontmatter/internal"
)

type Semver struct{ Major, Minor, Patch int }

func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

var VERSION = Semver{0, 1, 0}

type options struct {
	dryRun             bool
	includeHiddenFiles bool
}

const usage = `fm -- Markdown frontmatter batch editor

Usage:
  fm [query] [flags]

Query is read from stdin if omitted. Multiple queries may be separated
by ';' and '--' starts a line comment, so an SQL-style script can be
piped in:

  fm < script.sql

Flags:
  -h, --help      Show this message.
  -d, --dry-run   Simulate the operation without editing any files.
  -H, --hidden    Include hidden files (ignored by default).
`

func main() {
	opts, args, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	okOut := io.Writer(os.Stdout)
	errOut := io.Writer(os.Stderr)

	prog, err := parseProgram(opts, args, os.Stdin, okOut, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		os.Exit(2)
	}

	if ok := run(prog, opts, args, os.Stdin, okOut, errOut); !ok {
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

func parseProgram(opts options, args []string, in io.Reader, out, errOut io.Writer) (*internal.Program, error) {
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
	if len(args) > 0 {
		parts := make([]string, len(args))
		for i, arg := range args {
			if strings.ContainsAny(arg, " \t") {
				parts[i] = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
			} else {
				parts[i] = arg
			}
		}
		return strings.Join(parts, " "), nil
	}
	b, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
