<!-- baseline-sha: 36b91a49954be6c1bd800f4d16a46021b1f87ec6 -->

# Architecture Baseline

## Overview

`fm` is a command-line tool for querying and mutating YAML frontmatter in Markdown
files. It targets knowledge management workflows (e.g. Obsidian vaults) where
structured metadata sits in the `---` YAML block at the top of `.md` files.

The tool exposes an SQL-inspired DSL with three top-level statements:

```
select <exprs|*> from <globs> [where <expr>] [sort by <terms>] [limit <n>]
update <globs> set <assigns> [where <expr>]
alter  <globs> drop   <fields>  [where <expr>]
alter  <globs> rename <pairs>   [where <expr>]
```

Multiple statements may be combined into an SQL-style script separated by `;`,
with `--` line comments. Scripts can be read from stdin (`fm < script.sql`) or
passed as a single positional argument.

Module path: `github.com/nofuss-io/frontmatter`
Language: Go 1.23
Binary name: `fm`

The engine is also published as a reusable Go library: the root package
`frontmatter` re-exports the public types and entry points from `internal`
(`Program`, `Query`, `ExecOptions`, `Document`, `ReadDocument`, `Write`,
`ParseProgram`, `ParseQuery`, `ExpandGlobs`, `NewOutput`, `PrintTable`).

---

## Repository Layout

```
fm/
├── cmd/fm/                  # Application entrypoint (package main)
│   ├── main.go              # Flag parsing, completion/install-skill dispatch, Program.Run
│   ├── completion.go        # Static bash/zsh/fish completion scripts
│   └── install_skill.go     # `fm install-skill {claude|codex|copilot|gemini}` subcommand
├── internal/                # Core engine (package internal)
│   ├── ast.go               # Program / Query interface / SelectQuery / UpdateQuery / AlterQuery / Expr nodes / operators / FieldType / LiteralKind
│   ├── parse.go             # Recursive-descent cursor-based parser for the DSL
│   ├── parse_fuzz_test.go   # go test -fuzz fuzz target for ParseProgram
│   ├── eval.go              # Value model, Cast pipeline, expression eval, list/scalar comparisons, Assign.Apply
│   ├── eval_query.go        # Query.Eval implementations (select/update/alter projection + mutation)
│   ├── exec.go              # Program.Run orchestrator: glob plan, per-file loop, mutation write-back
│   ├── result.go            # Output / Table / TableRow result accumulator + sort/limit/short-circuit
│   ├── format.go            # Table building for regular + `select *` queries; delegates rendering to table.Renderer
│   ├── document.go          # ReadDocument / Write — `---` YAML block parse + emit
│   ├── file.go              # FrontMatter / Document / FilePath types, ExpandGlobs
│   ├── table/               # Pluggable table rendering (package table)
│   │   ├── table.go         # Table struct + Renderer interface
│   │   ├── simple.go        # Simple: tab-aligned with dashed separator (default)
│   │   ├── csv.go           # CSV: RFC 4180 CSV with header row
│   │   ├── markdown.go      # Markdown: GFM pipe-table
│   │   ├── full.go          # Full: box-drawing Unicode borders
│   │   └── *_test.go        # Renderer unit tests
│   └── *_test.go            # Unit tests: parse, eval_cast, eval_expr, eval_query, eval_types, file
├── docs/                    # Documentation + embedded assets (package docs)
│   ├── embed.go             # go:embed exposes SkillMD and SkillManualMD byte slices
│   ├── SKILL.md             # Claude Code skill (embedded, installed via `fm install-skill`)
│   ├── manual.md            # Full DSL reference
│   ├── fm.1.md              # Pandoc source for the top-level man page
│   ├── man/                 # Generated roff man pages (fm.1, …)
│   └── tutorial/            # Step-by-step walkthrough
│       ├── tutorial.md
│       ├── tutorial.sql     # SQL-style script variant
│       ├── tutorial.sh      # Shell script variant
│       └── recipes/         # Sample `.md` fixtures
├── frontmatter.go           # Root package: public re-exports of internal types/funcs
├── test/
│   ├── README.md
│   └── e2e/                 # Black-box golden tests against the compiled binary
│       ├── integration_test.go
│       ├── README.md
│       └── cases/<name>/    # input/, cmd, expected, expected_stderr, expected_exit
├── architecture/
│   ├── BASELINE.md          # This file
│   ├── FEATURE_WINDOWS.md   # Future feature: window/aggregate clauses
│   ├── PLAN_260525_oss_polish.md
│   └── TARGET_260525_oss_polish.md
├── .github/
│   ├── workflows/ci.yml         # lint + test on push/PR to main
│   ├── workflows/release.yml    # tag-triggered goreleaser + homebrew publish
│   ├── ISSUE_TEMPLATE/*         # bug report, feature request, question
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
├── .githooks/pre-commit     # gofmt staged Go, just lint, just test
├── .golangci.yml            # golangci-lint v2 config (errcheck/govet/staticcheck/…)
├── .goreleaser.yaml         # Multi-OS archives, checksums, Homebrew tap
├── .editorconfig
├── vendor/                  # Vendored Go dependencies (yaml.v3)
├── go.mod / go.sum
├── justfile                 # build, install, lint[-fix], test, vendor, dev, man, completions, check-links, release-check, release-snapshot, new-e2e-test, setup
├── CONTRIBUTING.md
├── CODEOWNERS
├── SECURITY.md
├── LICENSE                  # MIT
├── README.md
├── CLAUDE.md → AGENTS.md    # Agent guidance / repository map
└── .gitignore               # ignores fm binary, .obsidian/, docs/man/, completions/, Q.md
```

---

## Package Structure

Five Go packages:

| Package        | Path                  | Role |
|:---------------|:----------------------|:-----|
| `main`         | `cmd/fm/`             | CLI: flag parsing, `completion`/`install-skill` subcommands, `Version`/`Commit` ldflags init, calls `Program.Run` |
| `frontmatter`  | `./` (root)           | Public library API: thin re-export shell over `internal` |
| `internal`     | `internal/`           | Engine: AST, parser, evaluator, executor, formatter, file/document I/O |
| `table`        | `internal/table/`     | Pluggable table rendering: `Simple`, `CSV`, `Markdown`, `Full` renderers |
| `docs`         | `docs/`               | Embedded assets: `SkillMD` and `SkillManualMD` byte slices via `go:embed` |
| `integration`  | `test/e2e/`           | Black-box golden tests (package only used by `go test`) |

`cmd/fm` imports `frontmatter` (not `internal` directly) so the CLI exercises
the same public surface as third-party consumers.

---

## CLI Entry Point (`cmd/fm/main.go`)

### Flags

| Flag                   | Effect |
|:-----------------------|:-------|
| `-h`, `--help`         | Print usage and exit |
| `-V`, `--version`      | Print version and exit |
| `-d`, `--dry-run`      | Run all statements in memory; suppress writes to disk |
| `-s`, `--silent`       | Suppress all normal output (errors still go to stderr) |
| `-v`, `--verbose`      | Print affected fields after `update` or `alter` statements |
| `-H`, `--include-hidden` / `--hidden` | Include dot-prefixed files in glob expansion |
| `--max-columns N`      | Column cap for `select *` output (default `internal.DefaultMaxColumns` = 20) |
| `--format FORMAT`      | Output renderer: `simple` (default), `csv`, `markdown`, `full` |

Positional tokens before the first `-`-prefixed token are treated as the query
source; flags follow. `parseFlags` splits args at the first dash-prefixed token,
so query positional args can precede flags.

### Subcommands

- `fm completion {bash|zsh|fish}` — writes a static completion script to stdout (`completion.go`).
- `fm install-skill {claude|codex|copilot|gemini}` — copies `docs/SKILL.md` and `docs/manual.md`
  (embedded in the binary via `go:embed`) into `~/<agent-dir>/skills/fm/`. Replaces the former
  `just install-skill` target.

### Query input

`readProgramString` selects the source:

1. Exactly one positional arg → used verbatim.
2. Two or more positional args → error (`expected single query argument, got N`).
3. No positional args → read full program from stdin.

### Version / Commit

`Version` and `Commit` package-level vars default to `"dev"` / `""` and are
overridden by `-ldflags '-X main.Version=… -X main.Commit=…'` from `justfile`
and `.goreleaser.yaml`. `init` also falls back to `debug.ReadBuildInfo` so
`go install` builds carry the module version and the truncated VCS revision.

### Dispatch

`main` calls `fm.ParseProgram`, refuses an empty `Stmts`, then invokes
`prog.Run(ExecOptions{DryRun, Silent, Verbose, MaxColumns, IncludeHidden, Renderer}, stdout, stderr)`.
Run-level success determines exit code (0 = all files ok, 1 = at least one
per-file error, 2 = pre-execution failure).

---

## Engine (`internal/`)

### Data model (`internal/file.go`, `internal/document.go`)

```go
type FilePath = string

type FrontMatter map[string]any

type Document struct {
    FrontMatter FrontMatter
    Body        string
}
```

| Function | Purpose |
|:---------|:--------|
| `ReadDocument(path)` | Parse the leading `---\nYAML\n---\n` block; missing fence → body-only doc with empty frontmatter. Tolerates a closing `---` without trailing newline. |
| `Write(path, *Document)` | Re-emit `---\nYAML---\n` + body to disk (mode 0644) via `gopkg.in/yaml.v3` encoder at indent 2. |
| `ExpandGlobs(patterns)` | Resolve glob patterns and bare paths. Tokens with `*?[` go through `filepath.Glob`; non-regular matches are silently skipped. Bare-token tokens must exist or it errors. |

### AST (`internal/ast.go`)

```go
type Program struct { Stmts []Query }

type Query interface {
    Eval(fm FrontMatter) (*TableRow, error)
    IsMutation() bool
    Globs() []string
    q() query        // unexported: shared shape used by Output/Table
}

type query struct {
    Select []Expr
    Star   bool        // true when source had `select *`; Select is ignored
    From   []FilePath
    Where  Expr        // nil if absent
    SortBy []SortTerm
    Limit  int         // 0 = no limit
}

type SelectQuery = query

type UpdateQuery struct {
    query
    Set []Assign
}

type AlterQuery struct {
    query
    Op     AlterOp         // AlterDrop | AlterRename
    Drop   []Field
    Rename []RenamePair
}
```

Field / Assign / SortTerm / RenamePair, AssignOp (`= += -=`), AlterOp
(`Drop|Rename`) are unchanged from the previous baseline in spirit.

#### Expression nodes

| Node       | Shape                                | Notes |
|:-----------|:-------------------------------------|:------|
| `BinExpr`  | `{Op BinOp, Left, Right Expr}`       | Boolean, comparison, set-overlap, arithmetic |
| `UnaryExpr`| `{Op UnaryOp, Operand Expr}`         | `not`, arithmetic `-` |
| `FieldExpr`| `{Field Field}`                      | Field reference with optional type annotation |
| `LitExpr`  | `{Kind LiteralKind, Value string}`   | string / int / numeric / bool / null literal |
| `ListExpr` | `{Elems []Expr}`                     | Bracketed list literal `[e1, e2, …]`; evaluates element-wise to a `TypeList` value |

#### Operators

| Group     | Operators                                     |
|:----------|:----------------------------------------------|
| Boolean   | `or`, `and`, `not`                            |
| Compare   | `=`, `!=`, `<`, `<=`, `>`, `>=`, `<=>`        |
| Additive  | `+`, `-`                                      |
| Multiplicative | `*`, `/`                                 |
| Unary     | `-` (negation), `not`                         |

`<=>` (`BinOverlap`) is set overlap: scalar/scalar acts like `=`, list/list is
"share ≥1 element", scalar/list is membership.

#### Field type system

10 type tags (default `any`): `any`, `string`, `bool`, `int`, `number` (the
former `numeric`), `date`, `datetime`, `link`, `mdlink`, `list` (uniformly
list-of-string; an element-type annotation is a parse error).

Annotations appear after a colon: `tags:list`. Field names may be wrapped in
backticks to allow spaces or non-identifier characters.

### Parser (`internal/parse.go`)

Recursive-descent parser over an in-memory cursor (`*cursor{src []byte, pos int}`).
Public entry points (`ParseProgram`, `ParseQuery`, `ParseExpr`, and per-node
`Parse(io.Reader)` methods) `io.ReadAll` once into the cursor, then delegate
to cursor-based helpers.

| Function | Purpose |
|:---------|:--------|
| `ParseProgram(r)` | Read a sequence of `;`-separated statements. Tolerates leading/trailing/consecutive separators. `--` is a line comment. |
| `ParseQuery(r)`   | Parse a single statement; trailing input is allowed. |
| `ParseExpr(r)`    | Standalone expression. |
| `Field.Parse`, `Assign.Parse`, `SortTerm.Parse`, `RenamePair.Parse` | Per-node parsers (mostly used by tests). |

Each query type has a `parse(*cursor)` method dispatched from `parseOneQuery`.
Clause helpers (`readGlobs`, `readFieldList`, `readExprList`, `readSortTermList`,
`readAssignList`, `readRenamePairs`, `parseOptionalWhere`, `readIntLit`,
`expectKeyword`, `atStopKeyword`) share whitespace and stop-keyword handling so
each clause stops cleanly at the next SQL keyword or at `;`.

String literals support `'…' / "…" / """…""" / '''…''' / r"…" / R'…'` and the
usual C-style escapes (`\a \b \f \n \r \t \v \\ \' \" \` \?`, octal `\NNN`,
`\xHH`, `\uHHHH`, `\UHHHHHHHH`). Integers accept decimal and `0x…` hex; floats
have optional fraction/exponent; a leading `-` is folded into the literal when
followed by a digit or `.`, otherwise becomes `UnaryNeg`.

`validateLitAssign` rejects obviously impossible literal assignments at parse
time (e.g. `count:int = "hello"`); runtime cast failures handle the rest.

### Evaluator (`internal/eval.go`)

#### Value model

```go
type Value struct {
    Kind FieldType
    Data any
    Null bool
}
```

`Null` marks absence. It propagates through arithmetic and is falsey in
boolean context. A bare field reference acts as an existence check via
`truthy`.

#### Cast pipeline

`Cast(v, target)` dispatches on `target`:

- `any` → pass-through.
- `list` → wrap scalar, then cast every element to `string` (lists are
  uniformly list-of-string).
- Same-kind non-list → pass-through.
- One-element `list` → unwrap and recurse.
- Otherwise: per-pair `castTo<Bool|Int|Number|String|Date|Datetime|Link|MdLink>`.

Bool ↔ numeric conversion is restricted to `0`/`1`. Date strings must match
`2006-01-02`; datetime strings `2006-01-02T15:04:05`. `link`/`mdlink`
round-trip via `parseWikiLink` / `parseMdLink`. Casting `Null` is an error
(callers convert that error to a propagated null at the expression boundary).

#### Expression evaluation

| Node           | Behavior |
|:---------------|:---------|
| `LitExpr.Eval` | Parse raw text into a `Value` of the matching `Kind`; `null` → null. |
| `FieldExpr.Eval` | Look up `fm[name]` (null on absence); if typed, `Cast`; cast failure → null. |
| `ListExpr.Eval` | Evaluate every element; assemble a `TypeList` value. |
| `UnaryExpr.Eval` | `not` → `!truthy(v)`; `-` negates int/number, otherwise null. |
| `BinExpr.Eval`   | `and`/`or` short-circuit via `truthy`; arithmetic via `arith` (int stays int unless `/` has a remainder, then promotes to number; div-by-zero → null); comparisons via `compare` (lists routed to `compareList`). |

`scalarEq` compares same-kind values directly (with `time.Time.Equal` for
date/datetime), falling back to numeric and then string coercion.

List comparisons:

| Pattern               | Operator   | Semantics |
|:----------------------|:-----------|:----------|
| `list = list`         | `=` / `!=` | Set equality (same length, same multiset of elements) |
| `list <=> list`       | `<=>`      | Set overlap (share ≥1 element) |
| `list >= scalar`      | `>=`       | List contains scalar |
| `scalar <= list`      | `<=`       | List contains scalar (mirror) |
| `list <=> scalar` / `scalar <=> list` | `<=>` | Membership |
| anything else with a list operand and an ordering operator | `<`/`<=`/`>`/`>=` | `false` |

#### Assignments

| Method            | Behavior |
|:------------------|:---------|
| `Assign.Apply`    | Eval `Value`, cast to `Field.Type` (skip for `any`), then `=` / `+=` / `-=`. Cast-only form (no value) ensures the field exists (`nil` if absent) and recasts an existing value. |
| `applyListAdd`    | `+=` appends scalar(s) to a list. Nil current value is treated as empty list; non-list current value is promoted to a one-element list first. Duplicates (by `scalarEq`) are skipped. |
| `applyListSub`    | `-=` removes any element matching the supplied scalar(s) by `scalarEq`. A missing or non-list current value is a no-op. |

`anyFromValue` lowers `Value` back to the YAML-compatible `any` (recursive for
lists; `nil` for null). `valueFromAny` lifts the inverse direction; `time.Time`
with zero time components is classified as `date`, otherwise `datetime`.

### Query evaluation (`internal/eval_query.go`)

| Method | Behavior |
|:-------|:---------|
| `query.Eval` (select)      | Returns `(nil, nil)` if `where` is falsey/null; otherwise projects `Select` (or `Star`) into a `TableRow` and materializes `SortBy` keys. |
| `UpdateQuery.Eval`         | Where-check, then `Assign.Apply` each `Set` entry in order (first failure aborts), then project. `IsMutation() == true`. |
| `AlterQuery.Eval`          | Where-check, then `drop`/`rename`. For `drop`, projection happens *before* deletion so dropped values still appear in output. For `rename`, projection happens *after* so headers match new names. `IsMutation() == true`. |

### Executor (`internal/exec.go`)

```go
type ExecOptions struct {
    DryRun        bool
    Silent        bool             // redirect okOut to io.Discard
    Verbose       bool             // emit mutation rows (suppressed by default)
    MaxColumns    int              // 0 → DefaultMaxColumns (20)
    IncludeHidden bool
    Renderer      table.Renderer  // nil → table.Simple{}
}

func (Program) Run(opts ExecOptions, okOut, errOut io.Writer) (ok bool)
```

Lifecycle:

1. **`expandPlan`** — call `ExpandGlobs` once per statement; the union (in
   first-encounter order) drives the outer file loop. When `IncludeHidden` is
   false, dot-prefixed basenames are dropped *unless* they were named by a
   non-glob bare-path token.
2. **Per file**: `ReadDocument` once. Iterate statements; skip those whose path
   list doesn't contain the current file or whose `Output.Done` short-circuit
   has fired. Each surviving statement runs `Eval` against the in-memory
   frontmatter; output rows are routed to the per-statement `Table` via
   `Output.Append`. A mutation statement that succeeds marks the file dirty
   and (unless `Verbose`) its row is not appended; the first evaluation error
   halts further statements for that file and skips the write-back.
3. **Write-back** if the file was mutated, no statement halted, and `DryRun`
   is false.
4. **FROM-less selects** — after the file loop, any non-mutation statement
   whose glob list is empty is evaluated once against `FrontMatter{}` and its
   row is appended (no `filename` column). Enables expression evaluation
   without a file corpus, e.g. `fm 'select 1+1'`.
5. **`Output.Finalize`** sorts each Select table by `SortBy` (null sorts last;
   numeric kinds compare numerically; date kinds compare temporally; everything
   else falls back to `FormatValue` string compare) and truncates by `Limit`.
6. **`Output.Print`** writes each non-empty table to `okOut` in source order,
   separated by blank lines. Empty mutation tables (verbose=off) are silently
   skipped. Per-file errors go to `errOut` as `ERROR: <path>: <msg>`.

`Output.Done(i)` returns true once a non-mutation, no-sort statement has
collected `Limit` rows — this lets the file loop skip those statements (and
break out entirely when every table is `Done`).

### Result accumulator (`internal/result.go`)

```go
type Output struct { tables []*Table; errors io.Writer }
type Table  struct {
    sel        query
    mutation   bool
    noFile     bool              // true for FROM-less selects: filename column suppressed
    rows       []TableRow
    maxColumns int
    renderer   table.Renderer
}

type TableRow struct {
    path  FilePath
    print []Value           // populated for non-star selects
    star  map[string]Value  // populated for select *
    sort  []Value
}
```

`compareValues` provides the stable-sort ordering (null last, numeric numeric,
dates by `Before/After`, otherwise `FormatValue` string compare).

### Formatting (`internal/format.go`)

`format.go` builds `table.Table` structs (headers + `[][]string` rows) and
delegates rendering to the `table.Renderer` stored on each `Table`.

| Function / type          | Purpose |
|:-------------------------|:--------|
| `FieldName(e, idx)`      | Column header: field name for `FieldExpr`, else `expr<idx+1>`. |
| `FormatValue(v)`         | Plain string view (null → empty; lists → `[a, b, c]`). |
| `Value.String()`         | Debug-style `kind(data)` form, used by error messages and tests. |
| `Table.print`            | Builds `table.Table` (via `buildTable` or `buildStarTable`), calls `renderer.Render`, then prints `(N rows)` footer and hidden-column notice. |
| `Table.buildTable`       | Materializes non-star select: optional `filename` column + field columns. |
| `Table.buildStarTable`   | Alphabetical union of field names across rows, capped at `maxColumns`; returns hidden-column count. |
| `PrintTable(w, headers, rows, noFile)` | Convenience wrapper used by external callers; renders via `table.Simple{}`. |

### Table rendering (`internal/table/`)

```go
type Table struct {
    Headers []string
    Rows    [][]string
}

type Renderer interface {
    Render(table Table, w io.Writer) error
}
```

| Renderer    | Format |
|:------------|:-------|
| `Simple{}`  | Tab-aligned columns with a dashed separator row (default) |
| `CSV{}`     | RFC 4180 CSV with a header row |
| `Markdown{}`| GFM pipe-table with `|` delimiters and `:---` alignment row |
| `Full{}`    | Box-drawing Unicode borders (`┌─┬─┐ │ ├─┼─┤ └─┴─┘`) |

Selected via `--format {simple|csv|markdown|full}` at the CLI, resolved to a
`Renderer` in `cmd/fm/main.go:resolveRenderer` before `Program.Run` is called.

---

## External Dependencies

| Dependency          | Version | Role |
|:--------------------|:--------|:-----|
| `gopkg.in/yaml.v3`  | v3.0.1  | YAML parse/serialize for the `---` frontmatter block |
| `gopkg.in/check.v1` | (test-only, indirect via yaml.v3) | Not used by our code |

No CLI framework: argument and flag handling is hand-rolled with the standard
library `flag` package so positional query tokens can precede flags.

---

## Build, Test & Tooling

### Just targets (`justfile`)

| Target              | Action |
|:--------------------|:-------|
| `build`             | `go build -ldflags … -o fm ./cmd/fm` (Version/Commit injected from `git describe` / short HEAD) |
| `install`           | `go build` into `$GOPATH/bin/fm` |
| `lint` / `lint-fix` | `go fmt ./...` + `golangci-lint run [--fix] ./...` |
| `test FLAGS=""`     | `go test ./internal/... [FLAGS]` then `go test -count=1 ./test/... [FLAGS]` |
| `new-e2e-test NAME` | Scaffold a fresh case directory under `test/e2e/cases/<NAME>/` |
| `vendor`            | `go mod tidy && go mod vendor` |
| `dev *args`         | `go run ./cmd/fm <args>` |
| `man`               | `pandoc -s -t man docs/fm.1.md -o docs/man/fm.1` |
| `completions`       | Build, then write `completions/fm.{bash,zsh,fish}` |
| `check-links`       | `lychee` link-check over docs, README, architecture (requires `cargo install lychee`) |
| `release-check`     | `goreleaser check` |
| `release-snapshot`  | `goreleaser release --snapshot --clean` |
| `setup`             | Wire `core.hooksPath = .githooks`; install golangci-lint and lychee |

### Pre-commit hook (`.githooks/pre-commit`)

`gofmt -w` staged Go files, re-stage, then `just lint && just test`.
Activated via `just setup`.

### Linting (`.golangci.yml`)

golangci-lint v2 with `errcheck`, `govet`, `ineffassign`, `misspell`,
`staticcheck` (all checks except `-ST1000`), `unconvert`, `unused`. Formatters
`gofmt` and `goimports` (local prefix `github.com/nofuss-io/frontmatter`).

### Tests

- **Unit tests** in `internal/`: `parse_test.go`, `parse_fuzz_test.go`,
  `eval_test.go`, `eval_cast_test.go`, `eval_expr_test.go`,
  `eval_query_test.go`, `eval_types_test.go`, `file_test.go`.
- **End-to-end tests** in `test/e2e/`: `integration_test.go` builds `fm` once,
  copies each `cases/<name>/input/` into a `t.TempDir()`, runs `sh -ec` on
  `cmd`, and diffs `expected` / `expected_stderr` / `expected_exit` /
  `expected_files/*`. `-update` flag regenerates goldens.

### CI / Release (`.github/`)

- `workflows/ci.yml`: on push & PR to `main` — `just lint` then `just test`
  on Ubuntu, Go 1.23, with `extractions/setup-just` and golangci-lint.
- `workflows/release.yml`: on `v*.*.*` tag — install pandoc + just, then
  `goreleaser/goreleaser-action` with `args: release --clean`.
- `dependabot.yml`: keeps Go modules and GitHub Actions up to date.
- Issue/PR templates and `CODEOWNERS` under `.github/`.

### Goreleaser (`.goreleaser.yaml`)

`before.hooks` runs `just man` and `just completions`. Build matrix
linux/darwin/windows × amd64/arm64 (windows: amd64 only), CGO disabled,
`-s -w` plus `main.Version`/`main.Commit` ldflags. Archives bundle
`LICENSE`, `README.md`, `docs/manual.md`, `docs/man/fm.1`, and
`completions/*`. Changelog groups by Conventional Commits. Homebrew tap
publishes to `NoFuss-io/homebrew-tap` (Formula directory) and installs
the binary, the man page, and shell completions via
`generate_completions_from_executable`.

---

## Query Workflows

### Select

```
Program.Run → expandPlan → for each file in union:
    ReadDocument → for each select stmt whose globs include this file:
        Eval (where + project Select|Star → TableRow) → Output.Append
collect rows → Output.Finalize (sort + limit) → Output.Print
```

### Update

```
Program.Run → expandPlan → for each file in union:
    ReadDocument → for each update stmt:
        Eval (where + apply Set in order)
        → mark file dirty, Append projected row
    write doc back (unless DryRun or any stmt errored on this file)
```

### Alter

```
Program.Run → expandPlan → for each file in union:
    ReadDocument → for each alter stmt:
        Eval (drop or rename)
        → mark file dirty, Append projected row
            (drop projects pre-deletion values; rename projects post-rename names)
    write doc back (unless DryRun or any stmt errored on this file)
```

---

## Notable Design Decisions

- **Library + thin CLI split, with a public Go API.** The engine lives in
  `internal/` and is re-exported through the root `frontmatter` package, so
  third-party Go consumers get a stable surface without inheriting the CLI.
- **Single-pass multi-statement execution.** `Program.Run` reads each file once
  and runs every matching statement against the same in-memory frontmatter,
  with a single write-back per file. This makes scripts behave consistently
  and eliminates the previous "no `--dry-run` for multi-statement scripts"
  caveat at the CLI layer.
- **Halted statements skip the write.** Any per-file evaluation error aborts
  further statements for that file *and* cancels its write-back, so partially
  mutated frontmatter never reaches disk.
- **Short-circuit limit.** `Output.Done` lets a `select … limit N` (without
  `sort by`) stop consuming files as soon as it has enough rows; when every
  statement is done, the outer file loop exits early.
- **`select *` with a column cap.** Star selects materialize the union of
  field names across all matched files; `--max-columns` (default 20) caps the
  rendered width with a trailing "(N more column(s) hidden)" note.
- **Hidden files opt-in.** `filterHidden` drops dot-prefixed basenames from
  glob expansion by default; explicit bare-path tokens (`fm select … from .hidden.md`)
  bypass the filter. `-H/--hidden` re-enables them globally.
- **Single `any`-typed frontmatter map.** Every field is `any` in memory;
  types are enforced at query/mutation time via `Cast`.
- **Null as a first-class value.** Missing fields and failed casts collapse
  to the same `Null` sentinel so expressions stay composable.
- **Hand-rolled DSL parser.** Recursive descent over an in-memory byte cursor
  keeps the parser dependency-free, supports backtick-quoted identifiers, raw
  and triple-quoted strings, list literals, and shared stop-keyword handling
  without a separate lexer.
- **Pluggable table rendering.** `internal/table.Renderer` decouples output
  format from the rest of the engine. The four built-in renderers (`Simple`,
  `CSV`, `Markdown`, `Full`) are selected at the CLI layer via `--format` and
  injected through `ExecOptions.Renderer`; adding a new format requires only a
  new `Renderer` implementation and a `resolveRenderer` case.
- **Mutation output suppressed by default.** `update`/`alter` rows are not
  appended to the output table unless `--verbose` is passed. This keeps the
  default output clean while preserving the option to inspect what changed.
- **FROM-less SELECT (virtual document).** A `select` with no `FROM` clause
  evaluates once against an empty `FrontMatter{}`, enabling expression
  evaluation (e.g. `fm 'select 1+2'`) without touching any files.
- **Embedded skill assets.** `docs/embed.go` uses `go:embed` to bake
  `docs/SKILL.md` and `docs/manual.md` into the binary. `fm install-skill`
  writes them out to `~/<agent-dir>/skills/fm/`, so the binary is the single
  source of truth for skill installation across all supported AI agents.
- **Obsidian alignment.** Wikilink (`[[ref|title]]`) and Markdown-link
  (`[title](ref)`) are first-class field types with bidirectional casts;
  `.obsidian/` is git-ignored.
- **Distribution-ready.** goreleaser produces archived binaries for
  linux/darwin/windows (amd64+arm64), a Homebrew tap formula, and bundles
  man pages and shell completions generated from `docs/fm.1.md` and the
  `completion` subcommand.
