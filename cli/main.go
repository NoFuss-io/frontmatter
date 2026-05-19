package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/backlin/frontmatter/lib"
)

type Semver struct{ Major, Minor, Patch int }

func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

var VERSION = Semver{0, 1, 0}

type options struct {
	dryRun  bool
	silent  bool
	verbose bool
}

const usage = `fm — Markdown frontmatter batch editor

Usage:
  fm [query] [flags]

Query is read from stdin if omitted.

Flags:
  --dry-run   Simulate the operation without editing any files.
  --silent    Suppress all output.
  -v          After update or alter, run a select on the affected files
              and fields and print the result.
  -h, --help  Show this message.
`

func main() {
	opts, args, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	out := io.Writer(os.Stdout)
	errOut := io.Writer(os.Stderr)
	if opts.silent {
		out = io.Discard
		errOut = io.Discard
	}

	if err := run(opts, args, os.Stdin, out, errOut); err != nil {
		fmt.Fprintln(errOut, err)
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

func run(opts options, args []string, in io.Reader, out, errOut io.Writer) error {
	src, err := readQuery(args, in)
	if err != nil {
		return err
	}
	if strings.TrimSpace(src) == "" {
		return fmt.Errorf("empty query")
	}

	q, err := lib.ParseQuery(strings.NewReader(src))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	switch q := q.(type) {
	case lib.SelectQuery:
		return runSelect(q, opts, out, errOut)
	case lib.UpdateQuery:
		return runUpdate(q, opts, out, errOut)
	case lib.AlterQuery:
		return runAlter(q, opts, out, errOut)
	}
	return fmt.Errorf("unknown query type %T", q)
}

func readQuery(args []string, in io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	b, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
