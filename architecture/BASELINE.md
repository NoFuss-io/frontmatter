<!-- baseline-sha: 0b7b09b3645a094b0b06efc7f97c8428d5002825 -->

# Architecture Baseline

## Overview

`fm` is a command-line tool for querying and mutating YAML frontmatter in Markdown
files. It is aimed at knowledge management workflows (e.g. Obsidian vaults) where
structured metadata sits in the `---` YAML block at the top of `.md` files.

The tool exposes an SQL-inspired DSL with three top-level statements:

```
select <exprs> from <globs> [where <expr>] [sort by <terms>] [limit <n>]
update <globs> set <assigns> [where <expr>]
alter  <globs> drop   <fields>  [where <expr>]
alter  <globs> rename <pairs>   [where <expr>]
```

Multiple statements may be combined into an SQL-style script separated by `;`,
with `--` line comments. Scripts can be read from stdin (`fm < script.sql`) or
passed as positional arguments.

Module path: `github.com/nofuss-io/fm`
Language: Go 1.23
Binary name: `fm`

---

## Repository Layout

```
fm/
├── cli/                    # Application entrypoint (package main)
│   ├── main.go             # Flag parsing, script dispatch
│   ├── select.go           # runSelect — query + render
│   ├── update.go           # runUpdate — apply assignments, write
│   └── alter.go            # runAlter — drop/rename fields, write
├── lib/                    # Reusable library (package lib)
│   ├── ast.go              # Query/Expr/Field/Assign type definitions
│   ├── parse.go            # Recursive-descent parser for queries + expressions
│   ├── exec.go             # Value system, casts, expression eval, mutations
│   ├── file.go             # Frontmatter file read/write + glob expansion
│   ├── format.go           # Tab-aligned table rendering + row sorting
│   └── *_test.go           # Unit tests for parser, exec, casts, file I/O
├── docs/
│   ├── Manual.md           # Full DSL reference (syntax, semantics, edge cases)
│   ├── tutorial.md         # Step-by-step walkthrough
│   ├── tutorial_script.sql # Tutorial captured as a script
│   ├── tutorial_quoted.sh  # Tutorial as separate quoted `fm` calls
│   ├── tutorial_unquoted.sh# Tutorial without shell quoting (zsh-friendly)
│   ├── tutorial/           # Sample recipe `.md` files (manual fixtures)
│   └── man/                # Generated man pages (git-ignored)
├── architecture/
│   ├── BASELINE.md         # This file
│   ├── FEATURE_WINDOWS.md  # Future feature: window/aggregate clauses
│   ├── PLAN_260511_homebrew.md   # Homebrew distribution plan
│   └── TARGET_260511_homebrew.md # Homebrew distribution target state
├── vendor/                 # Vendored Go dependencies (only yaml.v3)
├── go.mod / go.sum         # Module definition and checksums
├── justfile                # Build automation (build, install, test, lint, dev)
├── README.md               # Quick start and examples
├── SKILL.md                # Claude Code skill describing how to drive `fm`
├── CLAUDE.md               # Agent guidance — points to AGENTS.md
├── AGENTS.md               # Repository map for AI agents
└── .gitignore              # Ignores fm binary, .obsidian/, docs/man/
```

---

## Package Structure

Two Go packages:

| Package | Path  | Role |
|:--------|:------|:-----|
| `main`  | `cli/` | Thin entry-point: flag parsing, file iteration, error reporting |
| `lib`   | `lib/` | Reusable library: AST, parser, evaluator, file I/O, formatting |

The CLI imports `lib` and contains no business logic of its own beyond glue
between flags, the parser, and the evaluator.

---

## CLI Entry Point (`cli/main.go`)

### Flags

| Flag | Effect |
|:-----|:-------|
| `--dry-run` | Simulate without writing. Rejected for multi-statement scripts (no transactional layer) |
| `--silent`  | Discard stdout and stderr |
| `-v`        | After `update`/`alter`, run an implicit `select` over affected files/fields |
| `-h`, `--help` | Print usage and exit |

### Query input

`readQuery` picks the source in this priority order:

1. Single positional arg → used verbatim.
2. Multiple positional args → joined with spaces; args containing whitespace are double-quoted.
3. No positional args → read the full query from stdin.

### Dispatch

`run` calls `lib.ParseProgram` on the source, then iterates `Stmts` through
`runStatement`, which type-switches on `lib.Query` (`SelectQuery`, `UpdateQuery`,
`AlterQuery`) and delegates to `runSelect`/`runUpdate`/`runAlter`.

Per-file errors (read failures, cast failures, write failures) are reported to
`errOut` and the loop continues. Multi-statement failure messages are prefixed
with the statement index.

### Verbose mode (`update`, `alter`)

After mutating, `printAffected` re-renders touched files as a select-style table
restricted to the fields that were assigned (update) or dropped/renamed (alter).
For renames, the *new* names are shown.

---

## Library: `lib/`

### Data model (`lib/file.go`)

```go
type FrontMatter map[string]any

type File struct {
    Path        string
    FrontMatter FrontMatter
    Body        string
}
```

| Function       | Purpose |
|:---------------|:--------|
| `ReadFile(path)` | Parse `---\nYAML\n---\n` header, leave the rest as `Body`. Handles a closing fence without trailing newline. Files without a leading `---\n` are treated as body-only with empty frontmatter. |
| `File.Write()` | Re-emit `---` YAML block + body to the original path (mode 0644). YAML is re-encoded with `gopkg.in/yaml.v3` at indent 2. |
| `ExpandGlobs(patterns)` | Resolve glob patterns and bare paths. Patterns with `*?[` go through `filepath.Glob`; bare tokens must exist or an error is returned. Non-regular files (directories, sockets) are silently skipped from glob matches. |

### AST (`lib/ast.go`)

```go
type Query interface { query() }

type SelectQuery struct {
    Fields []Expr
    From   []string
    Where  Expr        // nil if absent
    SortBy []SortTerm
    Limit  int         // 0 = no limit
}

type UpdateQuery struct {
    From  []string
    Set   []Assign
    Where Expr
}

type AlterQuery struct {
    From   []string
    Op     AlterOp     // AlterDrop | AlterRename
    Drop   []Field     // populated when Op == AlterDrop
    Rename []RenamePair// populated when Op == AlterRename
    Where  Expr
}

type Program struct { Stmts []Query }
```

#### Field, Assign, SortTerm, RenamePair

| Type         | Definition |
|:-------------|:-----------|
| `Field`      | `{Name string, Type FieldType}` — type annotation defaults to `any`. |
| `Assign`     | `{Field Field, Op AssignOp, Value Expr}` — `Value` is nil for the cast-only form (e.g. `set foo:int`). |
| `SortTerm`   | `{Expr Expr, Desc bool}` |
| `RenamePair` | `{From string, To string}` |
| `AssignOp`   | `OpSet (=) \| OpAdd (+=) \| OpSub (-=)` |
| `AlterOp`    | `AlterDrop \| AlterRename` |

#### Expressions

`Expr` is an interface implemented by four node types:

| Node       | Shape                       | Notes |
|:-----------|:----------------------------|:------|
| `BinExpr`  | `{Op BinOp, Left, Right Expr}` | Boolean, comparison, and arithmetic operators |
| `UnaryExpr`| `{Op UnaryOp, Operand Expr}` | `not` (boolean) or `-` (arithmetic negation) |
| `FieldExpr`| `{Field Field}`             | Field reference with optional type annotation |
| `LitExpr`  | `{Kind LiteralKind, Value string}` | String/int/numeric/bool/null literal |

Operators:

| Group     | Operators                            | Precedence (lowest → highest) |
|:----------|:-------------------------------------|:------------------------------|
| Boolean   | `or`, `and`, `not`                   | 1 |
| Compare   | `=`, `!=`, `<`, `<=`, `>`, `>=`      | 2 |
| Additive  | `+`, `-`                             | 3 |
| Multiplicative | `*`, `/`                        | 4 |
| Unary     | `-` (negation), `not`                | 5 |

Parentheses are accepted for grouping.

#### Field type system

10 type tags. The default when no annotation is given is `any`.

| Category   | Types                                                |
|:-----------|:-----------------------------------------------------|
| Wildcard   | `any` |
| Primitives | `string`, `bool`, `int`, `numeric`, `date`, `datetime` |
| Links      | `link` (`[[ref\|title]]`), `mdlink` (`[title](ref)`) |
| Compound   | `list` (list-of-string; element-type annotation is a parse error) |

Annotations appear after a colon: `tags:list`. Field names may be wrapped in
backticks to allow spaces or other non-identifier characters: ``select `Last modified` from *``.

### Parser (`lib/parse.go`)

Recursive-descent parser over an in-memory byte cursor. Public entry points
(`ParseProgram`, `ParseQuery`, `ParseExpr`, and the per-node `Parse(io.Reader)`
methods) `io.ReadAll` the input once, build a `cursor{src []byte, pos int}`,
then delegate to the cursor-based helpers. Entry points:

| Function | Purpose |
|:---------|:--------|
| `ParseProgram(r)` | Read a sequence of `;`-separated statements. Tolerates leading, trailing, and consecutive separators. `--` is a line comment treated as whitespace. |
| `ParseQuery(r)`   | Parse a single statement (no trailing-input check). |
| `ParseExpr(r)`    | Parse a standalone expression. |
| `Field.Parse`, `Assign.Parse`, `SortTerm.Parse`, `RenamePair.Parse` | Per-node parsers (used by tests). |

Each query type has a `parse(*cursor)` method invoked via the dispatch in
`parseOneQuery`. Clause helpers (`readGlobs`, `readFieldList`, `readExprList`,
`readSortTermList`, `readAssignList`, `readRenamePairs`, `parseOptionalWhere`,
`readIntLit`, `expectKeyword`, `atStopKeyword`) share whitespace and stop-keyword
handling so each clause stops cleanly at the next SQL keyword (`from`, `set`,
`drop`, `rename`, `where`, `sort`, `limit`, `by`) or at `;`.

#### String literals

`LitExpr.parseString` supports:

- Single (`'…'`) and double (`"…"`) quoted strings.
- Triple-quoted forms (`"""…"""`, `'''…'''`) for multi-line literals.
- Raw-string prefix (`r"…"` / `r'…'` / `R"…"`) disabling escape interpretation.
- C-style escape sequences: `\a \b \f \n \r \t \v \\ \' \" \` \?`, octal `\NNN`, `\xHH`, `\uHHHH`, `\UHHHHHHHH`.

#### Numeric literals

Integers (decimal and `0x…` hex) or floats (with optional fraction and exponent).
A leading `-` is folded into the literal when followed by a digit or `.`;
otherwise `-` becomes a `UnaryNeg`.

#### Static lit-vs-field validation

`validateLitAssign` rejects obviously impossible literal assignments at parse
time (e.g. `count:int = "hello"`). Runtime cast failures still fall through to
the executor for non-literal expressions.

### Evaluator (`lib/exec.go`)

#### Value model

```go
type Value struct {
    Kind FieldType
    Data any
    Null bool
}
type Row []Value
```

`Null` marks absence — missing fields or failed casts. Null propagates through
arithmetic and is falsey in boolean context. A bare field reference acts as an
existence check via `truthy`.

#### Cast pipeline

`Cast(v, target)` dispatches on `target`:

- `any` is pass-through.
- `list` always coerces every element to `string` (scalars are wrapped first).
  Lists are uniformly list-of-string.
- Same-kind non-list targets are pass-through.
- One-element `list` → scalar unwraps then recurses.
- Per-pair `castTo<Bool|Int|Number|String|Date|Datetime|Link|MdLink>` handles
  remaining cross-type conversions.

Bool ↔ int conversion is restricted to `0`/`1` to avoid surprises. Date strings
must match `2006-01-02`; datetime strings `2006-01-02T15:04:05`. `link` is
`[[ref]]` or `[[ref|title]]`; `mdlink` is `[title](ref)`; they round-trip
through each other via `parseWikiLink` / `parseMdLink`.

#### Expression evaluation

| Node                | Behavior |
|:--------------------|:---------|
| `LitExpr.Eval`      | Parse the raw `Value` string to its `Kind`. `null` → null `Value`. |
| `FieldExpr.Eval`    | Look up `fm[name]`; absent → null. If a type annotation is present, runs `Cast`; cast failure → null. |
| `UnaryExpr.Eval`    | `not` returns `bool(!truthy(v))`; `-` negates int/number, nulls everything else. |
| `BinExpr.Eval`      | `and`/`or` short-circuit-style on `truthy(...)`; arithmetic in `arith` (int stays int unless division has a remainder, then it promotes to number; div-by-zero → null); comparisons in `compare` (list cases routed to `compareList`). |

`scalarEq` compares same-kind values directly (with `time.Time.Equal` for
dates/datetimes), falling back to numeric and then string coercion across
kinds.

List comparisons:

| Pattern              | Operator | Semantics |
|:---------------------|:---------|:----------|
| `list = list`        | `=`/`!=` | Set equality (same length, same multiset of elements) |
| `list >= scalar`     | `>=`     | List contains scalar |
| `scalar <= list`     | `<=`     | List contains scalar (mirror form) |
| Anything else with a list operand | ordering | `false` |

#### Mutations

| Method            | Behavior |
|:------------------|:---------|
| `Assign.Apply`    | Eval `Value`, cast to `Field.Type` (if not `any`), then write/add/subtract. Cast-only form (no value) ensures the field exists (`nil` if absent) and recasts an existing value. |
| `applyListAdd`    | `+=` appends scalar(s) to a list. A `nil` current value is treated as an empty list (no leading `null` element); a non-list, non-nil current value is promoted to a one-element list first. |
| `applyListSub`    | `-=` removes any element matching the supplied scalar(s) by `scalarEq`. A missing or `nil` current value is a no-op. |
| `UpdateQuery.Apply`, `AlterQuery.Apply`, `SelectQuery.Eval` | Top-level driver methods; honor an optional `Where` via `truthy`. |

`anyFromValue` lowers `Value` back to the `any` form used by the YAML map
(recursive for lists; `nil` for null).

`valueFromAny` lifts raw YAML values into `Value`. `time.Time` with all-zero
time components is classified as `date`; otherwise `datetime`.

### Formatting (`lib/format.go`)

| Function | Purpose |
|:---------|:--------|
| `FieldName(e, idx)` | Column header: field name for `FieldExpr`, else `expr<idx+1>`. |
| `FormatValue(v)` | Plain string view of a `Value` for tables. Null → empty; lists rendered as `[a, b, c]`. |
| `PrintTable(w, headers, paths, rows)` | Tab-writer output with a `filename` column prepended and a dashed separator row. |
| `SortRows(paths, rows, terms, fms)` | In-place stable sort by `SortTerm` evaluations. Null sorts last; numerics compare numerically; dates use `time.Time.Before/After`; everything else falls back to `FormatValue` string compare. |

---

## Query Workflows

### Select

```
ExpandGlobs → for each path:
    ReadFile → SelectQuery.Eval (where + project Fields → Row)
    drop rows that the where-clause filtered out
collect rows → SortRows (if sort-by) → trim (if limit) → PrintTable
```

### Update

```
ExpandGlobs → for each path:
    ReadFile → optional Where truthiness check
    applyAssignments (each Assign.Apply in order; first failure halts file)
    File.Write (skipped on --dry-run)
collect touched paths → printAffected (if -v)
```

### Alter

```
ExpandGlobs → for each path:
    ReadFile → optional Where truthiness check
    AlterQuery.Apply (drop fields or rename pairs)
    File.Write (skipped on --dry-run)
collect touched paths → printAffected (if -v)
```

For `alter rename` in verbose mode, the projection uses the *new* field names so
the output reflects post-mutation state.

---

## External Dependencies

| Dependency          | Version | Role |
|:--------------------|:--------|:-----|
| `gopkg.in/yaml.v3`  | v3.0.1  | YAML parse/serialize for the `---` frontmatter block |
| `gopkg.in/check.v1` | (test-only, indirect via yaml.v3) | Not used by our code |

No CLI framework: argument and flag handling is hand-rolled with `flag` from
the standard library so positional query tokens can precede flags.

---

## Build & Tooling

| Tool        | Config     | Targets |
|:------------|:-----------|:--------|
| Just        | `justfile` | `build`, `install`, `install-skill`, `lint`, `test`, `vendor`, `dev`, `help` |
| Go          | `go.mod`   | `go build ./cli`, `go run ./cli`, `go test ./...` |
| staticcheck | (via `lint`) | Static analysis on top of `go fmt` + `go vet` |

`install-skill` installs the binary to `$GOPATH/bin/fm` and then copies
`SKILL.md` to `~/.claude/skills/fm/SKILL.md` so Claude Code can drive `fm`
through a registered skill.

Tests live alongside source in `lib/`: `parse_test.go`, `exec_test.go`,
`exec_cast_test.go`, `file_test.go`. The recipe files under `docs/tutorial/`
act as fixtures for manual smoke-testing of the tutorial (reset with
`git restore docs/tutorial`).

No CI/CD pipeline is configured.

---

## Notable Design Decisions

- **Library + thin CLI split.** Everything that's testable (parsing, casts,
  evaluation, file I/O, formatting) lives in `lib`. The `cli` package only
  wires flags to library calls — keeps unit tests focused and lets external
  callers reuse the engine without inheriting the CLI.
- **Single `any`-typed frontmatter map.** Flexibility wins over compile-time
  type safety: every field is `any` in memory, with types enforced at
  query/mutation time via `Cast`.
- **Null as a first-class value.** Missing fields and failed casts collapse to
  the same `Null` sentinel rather than `nil`/error pairs, so expressions stay
  composable: `where missing_field + 1 > 0` quietly returns false instead of
  exploding.
- **No transactional layer for scripts.** Each statement re-reads files from
  disk. `--dry-run` is rejected for multi-statement scripts because skipped
  writes would make later statements observe stale state.
- **Hand-rolled DSL parser.** Recursive descent over an in-memory byte cursor keeps the
  parser dependency-free, supports backtick-quoted identifiers, raw and
  triple-quoted strings, and lets clause helpers share stop-keyword handling
  without a separate lexer.
- **Obsidian alignment.** Wikilink (`[[ref|title]]`) and Markdown-link
  (`[title](ref)`) are first-class field types with bidirectional casts.
  `.obsidian/` is git-ignored.
