# Baseline Architecture

## Commands (current)

| Command | Syntax | Notes |
|---------|--------|-------|
| `list`  | `list <glob\|files...> [<field\|comparison>...]` | field args = columns; comparisons filter rows |
| `set`   | `set <glob\|files...> <field=value>...` | variadic assignments |
| `rm`    | `rm <glob\|files...> <field>` | single field only |
| `cast`  | `cast <glob\|files...> <field> <type>` | type as separate arg |
| `check` | `check <glob\|files...> <field>` | validates type; exits non-zero on failure |

Global `--filter <expression>` flag applies to all commands before command-level filtering.

## Core types (core.go)

- `FieldType` — enum: Any, String, Int, Number, Date, Link, List
- `Field` — `{Name, Type, ElemType}` — ElemType non-nil for `list:<type>`
- `Comparison` — `{Field, Value *string}` — nil Value = type-only match
- `Expression` — `[][]Comparison` — outer slice = OR groups, inner = AND group

## Key behaviours

- **Arg splitting**: `splitArgs` separates file/glob args from field args by attempting glob expansion and stat; first non-matching arg starts the field tail.
- **Expression parsing**: string-split on `||` then `&&`; strips one layer of outer parens; no negation support.
- **Comparison matching**: `matchesCmp` checks field existence, optional type check, optional value equality via `fmtValue`.
- **Cast**: converts via string representation; supports int, number, date (multiple layouts), link (wraps in `[[]]`), list (wraps scalar in slice).
- **Date handling**: `dateVal` type preserves YAML `!!timestamp` tag to avoid quoted output.
- **Write**: always emits `---\n` frontmatter block; creates one if missing.

## Missing relative to TARGET.md

- SQL-style positional keywords (`from`, `set`, `drop`, `where`) — currently uses `--filter` flag and positional heuristics
- Commands renamed: `list` → `select`, `set`+`cast` → `update`, `rm` → `alter ... drop`; `check` removed
- `update` must unify set and cast: field-only arg = cast, `field=value` = set
- `+=` / `-=` operators (arithmetic on numbers, append/remove on lists)
- `!` negation on comparisons in expressions
- `rm` is not variadic; all commands should accept multiple fields
- `list:<type>` ElemType not used in `isType` or `castValue`
