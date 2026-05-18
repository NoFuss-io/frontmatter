# Eval implementation order

Goal: each AST type gets an eval method. Expressions return a typed value;
queries mutate a `Document` in place (or, for select, produce a projected row).

## API sketch

```go
// Value is the runtime value of an expression. Void means missing or cast
// failed; comparisons propagate Void as falsey, arithmetic as Void.
type Value struct {
    Kind FieldType
    Data any  // string, int64, float64, bool, time.Time, []Value, etc.
    Void bool
}

// Expressions: pure, no error — errors become Void per Manual.md.
type Expr interface {
    Eval(doc *Document) Value
}

// Assign: mutates doc.FrontMatter; error halts current file per Manual.md.
func (a *Assign) Apply(doc *Document) error

// Per-document query eval. SelectQuery returns a projected Row; the mutating
// queries follow the Apply convention used by Assign.
func (q *SelectQuery) Eval(fm *FrontMatter) (Row, error) // Row = []Value
func (q *UpdateQuery) Apply(fm *FrontMatter) error
func (q *AlterQuery)  Apply(fm *FrontMatter) error
```

Multi-document orchestration (glob expansion, sort across docs, limit) wraps
per-doc eval at a higher layer and is intentionally out of scope here.

## Ranking

| # | Target                       | Complexity | Why                                                                |
|--:|------------------------------|------------|--------------------------------------------------------------------|
| 1 | Value type + cast helpers    | Simple     | Foundational; pure functions over FieldType; no Document needed    |
| 2 | LitExpr.Eval                 | Trivial    | Parse Value string into typed Go value by Kind                     |
| 3 | FieldExpr.Eval               | Simple     | Map lookup + optional cast via #1                                  |
| 4 | UnaryExpr.Eval               | Simple     | Recursive eval + neg/not; void propagation                         |
| 5 | BinExpr.Eval (and/or)        | Simple     | Short-circuit boolean; void-as-falsey                              |
| 6 | BinExpr.Eval (arith)         | Medium     | +, -, *, /; numeric coercion; void propagation                     |
| 7 | BinExpr.Eval (comparison)    | Medium     | Type relaxation; list set-membership; equality vs ordering         |
| 8 | Assign.Apply                 | Medium     | Eval RHS, cast to field type, mutate map; +=/-= list & scalar      |
| 9 | SortTerm.Eval                | Trivial    | Wraps Expr.Eval; used by external sort comparator                  |
|10 | AlterQuery.Apply             | Medium     | Drop/rename map entries; optional where filter via #5–7            |
|11 | UpdateQuery.Apply            | Medium     | Where filter; iterate Assign list via #8                           |
|12 | SelectQuery.Eval (per-doc)   | Hard       | Where filter; project Fields list into Row; no sort/limit here     |

Build order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12

## Commits

- Tests for all 12 rows landed first as a single commit `eval tests`.
- Each implementation row commits separately: `eval task 1` … `eval task 12`.

## Scope

Single-document execution only. The runner that expands globs and iterates
documents (sequentially or in parallel) lives outside `lib/exec.go` and is
not covered by this plan. Sort/limit across documents likewise.
