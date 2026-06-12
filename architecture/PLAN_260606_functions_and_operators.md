baseline = ac32805229e40a4388c1978b1c51653159a873e0

# Plan: Functions and New Operators

Each phase ends with `just lint` and `just test`, edit till it passes, then commit and proceed to next phase.

---

## Phase 1 — AST

- [ ] **1.1** `internal/ast.go`: rename `BinOverlap` → `BinIntersect`; add `BinUnion`, `BinLike`, `BinNotLike`, `BinILike`, `BinNotILike`, `BinRegexp`, `BinNotRegexp` to `BinOp` iota.
- [ ] **1.2** `internal/ast.go`: add `FuncExpr` node:
  ```go
  type FuncExpr struct { Name string; Args []Expr }
  func (FuncExpr) expr() {}
  ```
  Add `Eval(fm FrontMatter) Value` stub (returns null) — filled in Phase 3.

---

## Phase 2 — Parser

- [ ] **2.1** `internal/parse.go` `parsePrimary`: after reading an ident that is not a keyword or field-type, peek for `(`; if present, consume args as a comma-separated `readExprList` until `)` and return `FuncExpr{Name: strings.ToLower(word), Args: args}`.
- [ ] **2.2** `internal/parse.go` `parseComparison`: after reading `left`, check for infix keyword operators before the symbol operators:
  - `NOT LIKE` / `NOT ILIKE` / `NOT REGEXP` (two-token) → consume both, parse right, return `BinExpr`.
  - `LIKE` / `ILIKE` / `REGEXP` (single-token) → consume, parse right, return `BinExpr`.
  - Keep existing symbol operators (`<=>` now maps to `BinIntersect`, add `>=<` → `BinUnion`).
- [ ] **2.3** `internal/parse_test.go`: add parse tests for function calls (`LOWER("x")`, zero-arg `TODAY()`, vararg `CONCAT("a","b","c")`) and new operators (`LIKE`, `ILIKE`, `NOT REGEXP`, `<=>`, `>=<`).

---

## Phase 3 — Evaluator: new operators

- [ ] **3.1** `internal/eval.go` `BinExpr.Eval`: route `BinIntersect` and `BinUnion` to new `setOp` helper (returns `TypeList`); route `BinLike/NotLike/ILike/NotILike/Regexp/NotRegexp` to new `matchOp` helper.
- [ ] **3.2** `internal/eval.go`: implement `setOp(op, l, r Value) Value`:
  - Coerce both sides to `[]Value` (scalar → single-element list).
  - `BinIntersect`: order-preserving intersection by `scalarEq`.
  - `BinUnion`: order-preserving union (left first, then right elements not already in left).
- [ ] **3.3** `internal/eval.go`: implement `matchOp(op, l, r Value) Value`:
  - Coerce both sides to string via `Cast(…, TypeString)`; null → return null.
  - `LIKE`/`ILIKE`: convert SQL wildcard pattern (`%` → `.*`, `_` → `.`, escape other regex meta-chars) then `regexp.MatchString`.
  - `ILIKE`: compile with `(?i)` prefix.
  - `REGEXP`/`NOT REGEXP`: compile pattern via Go's `regexp.MatchString` (RE2-compatible subset; no external dep).
  - `Not` variants negate the bool result.
- [ ] **3.4** `internal/eval_expr_test.go`: add table-driven tests for `<=>` (intersection), `>=<` (union), `LIKE`, `ILIKE`, `NOT LIKE`, `REGEXP`, `NOT REGEXP`.

---

## Phase 4 — Evaluator: functions

- [ ] **4.1** Create `internal/eval_func.go` with `(e FuncExpr) Eval(fm FrontMatter) Value` dispatching on `e.Name` to per-function helpers. Unknown names return null.
- [ ] **4.2** Implement string functions: `lower`, `upper`, `length`, `substr`, `starts_with`, `ends_with`, `contains_substr`, `trim`, `ltrim`, `rtrim`, `replace`, `split`, `concat`, `regexp_contains`, `regexp_extract`, `to_string`.
- [ ] **4.3** Implement numeric functions: `abs`, `ceil`, `floor`, `round`, `mod`, `sqrt`, `pow`, `greatest`, `least`, `coalesce`.
- [ ] **4.4** Implement list functions: `array_length`, `array_contains`, `array_concat`, `distinct`, `array_to_string`.
- [ ] **4.5** Implement date functions: `today`, `year`, `month`, `day`, `date_diff`.
- [ ] **4.6** Create `internal/eval_func_test.go` with table-driven tests for every function (happy path, null propagation, arity mismatch).
