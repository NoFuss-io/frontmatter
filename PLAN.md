# Implementation Plan

## 1. Separate `Comparison` and `Assignment` types (core.go)

`+=`/`-=` are assignment-only; `!` negation is comparison-only. Two distinct types:

```go
// Used in `where` clauses
type Comparison struct {
    Neg   bool
    Field Field
    Value *string // nil = type-only match
}

// Used in `update set` clauses
type AssignOp int
const (OpSet AssignOp = iota; OpAdd; OpSub)

type Assignment struct {
    Field Field
    Op    AssignOp
    Value *string // nil = cast to field.Type
}
```

Update `ParseComparison` to strip leading `!` and set `Neg`.
Add `ParseAssignment` to detect `+=` and `-=` before `=`.

Update `matchesCmp` to honour `Neg` and list containment: when the field value is a slice and `Value != nil`, check whether the slice contains the value rather than comparing the whole field.

## 2. Add `Apply` method (core.go)

`File.Apply(a Assignment) error`
- `a.Value == nil`: cast field to `a.Field.Type` — reuse `castValue`; skip if current type doesn't match `a.Field.Type` (unless TypeAny)
- `OpSet`: set value string as-is (reuse `Set`)
- `OpAdd`: int/number → parse and add; string → append; list → append value if not present
- `OpSub`: int/number → parse and subtract; list → remove value if present

## 3. Replace `splitArgs` with keyword-based parsing (main.go)

Add helper:
```go
func splitOn(args []string, keyword string) (before, after []string, found bool)
```
Scans for the first occurrence of `keyword` and returns the two halves.

Each command parses its args as:
- `select`: fields (before `from`) / globs (after `from`, before `where`) / expression (after `where`)
- `update`: globs (before `set`) / field|assignment args (after `set`, before `where`) / expression (after `where`)
- `alter`:  globs (before `drop`) / fields (after `drop`, before `where`) / expression (after `where`)

Globs are expanded inline (reuse existing glob + stat logic, extracted from `splitArgs`).
Expression tokens after `where` are joined with a space and passed to `ParseExpression`.

## 4. Rewrite commands (main.go)

| Old | New | Change |
|-----|-----|--------|
| `list` | `select` | rename; use keyword parsing |
| `set` + `cast` | `update` | merge; field-only arg = cast; `field=value` = set; `+=`/`-=` supported |
| `rm` | `alter … drop` | rename; variadic fields; use keyword parsing |
| `check` | *(removed)* | delete |

Remove `--filter` persistent flag from root. Each command reads its own `where` clause.

## 5. Fix `list:<type>` element-type handling (core.go)

- `isType`: when `t == TypeList` and `field.ElemType != nil`, check each element matches the elem type.
- `castValue`: when targeting `TypeList` with `ElemType`, cast each element to elem type after wrapping.

## Order of work

1. Step 1 — extend `Comparison` (isolated, no command changes)
2. Step 2 — add `File.Update` (isolated, testable)
3. Step 5 — fix elem-type handling (isolated)
4. Step 3 — keyword parser helper
5. Step 4 — rewrite commands (depends on 1–4)
