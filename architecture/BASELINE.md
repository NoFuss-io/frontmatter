# Architecture Baseline

## Overview

`fm` is a command-line tool for querying and mutating YAML frontmatter in Markdown files.
It is aimed at knowledge management workflows (e.g. Obsidian vaults) where structured
metadata sits in the `---` YAML block at the top of `.md` files.

The tool exposes a SQL-inspired syntax:

```
fm select [fields] from <files> [where <expr>] [sort by <field> [desc]] [limit <n>]
fm update <files> set <assignments> [where <expr>]
fm alter <files> drop <fields> [where <expr>]
```

Module path: `github.com/backlin/frontmatter`  
Language: Go 1.22.2  
Binary name: `fm`

---

## Repository Layout

```
fm/
├── cli/                  # All application source (single package main)
│   ├── main.go           # Command definitions and top-level handlers
│   ├── core.go           # Data model, file I/O, matching, mutation
│   └── parse.go          # Parsers for fields, comparisons, assignments, expressions
├── docs/
│   ├── syntax.md         # Full CLI syntax reference
│   ├── tutorial.md       # Step-by-step walkthrough
│   └── tutorial/         # Sample Markdown recipe files used by tutorial
├── architecture/
│   └── BASELINE.md       # This file
├── vendor/               # Vendored Go dependencies
├── go.mod / go.sum       # Module definition and checksums
├── justfile              # Build automation (Just)
├── README.md             # Project overview and quick start
└── .gitignore            # Ignores fm binary and .obsidian metadata
```

---

## Package Structure

All source lives in a single Go package (`package main`) under `cli/`. There is no library
separation; the tool is a pure CLI binary.

---

## Commands

| Command  | Purpose |
|:---------|:--------|
| `select` | Print a table of frontmatter fields from matching files |
| `update` | Batch-apply field assignments (set, arithmetic, list append/remove) |
| `alter`  | Drop fields from matching files |

### select

```
fm select [<fields>] from <glob...> [where <expr>] [sort by <field> [desc]] [limit <n>]
```

- Outputs a tab-separated table (filename + requested fields).
- Fields may carry a type annotation (`field:type`) to filter by type.
- With no field list, only filenames are printed.

### update

```
fm update <glob...> set <assignments> [where <expr>]
```

Assignments:
- `field=value` — set field to value
- `field:type` — cast field to type
- `field+=value` — add number or append to list
- `field-=value` — subtract number or remove from list

### alter

```
fm alter <glob...> drop <fields> [where <expr>]
```

Removes listed fields from frontmatter; type-annotated fields are only removed if the
current value matches the specified type.

---

## Core Data Model (`cli/core.go`)

### `File`

```go
type File struct {
    Path string
    FM   map[string]any
    Body string
    hasFM bool
}
```

The central entity. `FM` is the parsed YAML frontmatter as a generic map.

### Key functions

| Function       | Signature                                      | Purpose |
|:---------------|:-----------------------------------------------|:--------|
| `ReadFile`     | `(path string) (*File, error)`                 | Parse frontmatter + body from disk |
| `Write`        | `(f *File) error`                              | Serialize frontmatter + body back to disk |
| `Matches`      | `(f *File, expr Expression) bool`              | Evaluate WHERE expression (OR of ANDs) |
| `Apply`        | `(f *File, a Assignment) error`                | Execute a single assignment operation |
| `Remove`       | `(f *File, field Field) bool`                  | Conditionally delete a field |
| `isType`       | `(v any, f Field) bool`                | Type-check a value against a Field spec |
| `castValue`    | `(v any, f Field) (any, error)`| Convert a value to the target type |
| `fmtValue`     | `(v any) string`                       | Render any value as a string |

### Date handling

Dates are stored and round-tripped as `!!timestamp` YAML scalars (date-only, no time
component) using the custom `dateVal` type and a bespoke YAML marshaller.

---

## Type System (`cli/parse.go`, `cli/core.go`)

17 type identifiers are supported:

| Category | Types |
|:---------|:------|
| Primitives | `any`, `string`, `bool`, `int`, `number`, `date` |
| Compound | `list`, `list:<elem-type>` |
| Obsidian | `link` (`[[…]]` wikilinks) |

Types are expressed in CLI arguments as `fieldname:type`, e.g. `tags:list:string`.

---

## Parsing (`cli/parse.go`)

### AST types

| Type         | Definition                         | Purpose |
|:-------------|:-----------------------------------|:--------|
| `Field`      | `{Name string, Type, ElemType *FieldType}` | Field name + optional type annotation |
| `Comparison` | `{Neg bool, Field, Value *string}` | Single WHERE predicate |
| `Assignment` | `{Field, Op AssignOp, Value string}` | Single SET operation |
| `Expression` | `[][]Comparison`                   | WHERE clause: outer slice = OR groups, inner = AND groups |
| `AssignOp`   | `OpSet \| OpAdd \| OpSub`          | Operator kind for assignments |

### Parsing functions

| Function            | Input example              | Output |
|:--------------------|:---------------------------|:-------|
| `ParseField`        | `tags:list:string`         | `Field{Name:"tags", Type:list, ElemType:string}` |
| `ParseComparison`   | `-status=draft`            | `Comparison{Neg:true, Field:"status", Value:"draft"}` |
| `ParseAssignment`   | `score+=1`                 | `Assignment{Field:"score", Op:OpAdd, Value:"1"}` |
| `ParseExpression`   | `a=1 or b=2 and c=3`       | `Expression{{a=1},{b=2,c=3}}` |

Boolean operators `or` / `and` are handled as keywords; `and` binds tighter (AND-within-OR).

---

## CLI Entry Point (`cli/main.go`)

### Argument parsing helpers

| Function        | Purpose |
|:----------------|:--------|
| `splitOn`       | Split token list on keyword boundaries (`from`, `set`, `drop`, `where`, `sort`, `limit`) |
| `splitCommas`   | Split comma-separated tokens, returning `[][]string` groups |
| `expandGlobs`   | Expand file glob patterns to concrete paths |
| `loadFiles`     | Read files from paths, apply optional WHERE filter |
| `writeErr`      | Report write errors to stderr without stopping batch |

---

## External Dependencies

| Dependency | Version | Role |
|:-----------|:--------|:-----|
| `github.com/spf13/cobra` | v1.10.2 | CLI framework (commands, help, completion) |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parse/serialize with node-level control |
| `github.com/spf13/pflag` | v1.0.9 | POSIX flag parsing (indirect, via Cobra) |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | Windows console fix (indirect, via Cobra) |

---

## Build & Tooling

| Tool     | Config      | Key targets |
|:---------|:------------|:------------|
| Just     | `justfile`  | `build`, `install`, `lint`, `vendor`, `dev`, `shell-completion` |
| Go       | `go.mod`    | `go build ./cli`, `go run ./cli` |
| staticcheck | (via lint) | Static analysis on top of `go vet` |

No CI/CD pipeline is configured. No automated tests (`*_test.go`) exist; the tutorial
files in `docs/tutorial/` serve as manual test fixtures (reset with `git reset --hard`).

---

## Workflows

### Query workflow

```
glob expansion → ReadFile (parse FM + body) → Matches (WHERE filter)
→ fmtValue (render) → tabwriter output
```

### Mutation workflow

```
glob expansion → ReadFile → Matches (WHERE filter)
→ Apply / Remove (mutate FM map) → Write (serialize back to disk)
```

---

## Notable Design Decisions

- **Single binary, no library**: All logic in `package main`; no exported API surface.
- **Generic frontmatter map**: `FM` is `map[string]any`, giving flexibility at the
  cost of type-safety; types are checked/enforced at query/mutation time.
- **Date-only timestamps**: Custom `dateVal` marshaller ensures dates are stored without
  time-zone noise (`2024-01-15` not `2024-01-15T00:00:00Z`).
- **Cobra shell completion**: Available for bash/zsh/fish via `fm completion <shell>`.
- **Obsidian compatibility**: `.gitignore` excludes `.obsidian/`; link type supports
  `[[wikilink]]` syntax.
