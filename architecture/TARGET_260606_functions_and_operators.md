baseline = ac32805229e40a4388c1978b1c51653159a873e0

# Target: Functions and New Operators

Add built-in function calls and new infix operators to the expression layer.

---

## New AST node: `FuncExpr`

```go
type FuncExpr struct {
    Name string   // lower-cased at parse time
    Args []Expr
}
func (FuncExpr) expr() {}
```

Parsed as `name(arg, …)` in `parsePrimary`: when an identifier is followed
immediately by `(`, it becomes a `FuncExpr` rather than a `FieldExpr`.

---

## New infix operators

### Set operators (return `TypeList`)

| Token | Const       | Semantics                                    |
|:------|:------------|:---------------------------------------------|
| `<=>` | `BinIntersect` | Set intersection; replaces `BinOverlap` (breaking) |
| `>=<` | `BinUnion`  | Set union (distinct, order-preserving)       |

`BinOverlap` is removed. The boolean overlap test is now expressible via
`ARRAY_LENGTH(a <=> b) > 0` or `ARRAY_CONTAINS(a, elem)`.

### String-matching operators (return `TypeBool`, infix, comparison level)

| Token      | Const        | Semantics                                        |
|:-----------|:-------------|:-------------------------------------------------|
| `LIKE`     | `BinLike`    | SQL wildcard: `%` = any substring, `_` = one char |
| `NOT LIKE` | `BinNotLike` | Negated LIKE                                     |
| `ILIKE`    | `BinILike`   | Case-insensitive LIKE                            |
| `NOT ILIKE`| `BinNotILike`| Negated ILIKE                                    |
| `REGEXP`   | `BinRegexp`  | Go `regexp` full-string match                    |
| `NOT REGEXP`| `BinNotRegexp`| Negated REGEXP                                  |

> **Regexp engine:** Use Go's standard `regexp` package throughout — for both
> the `REGEXP`/`NOT REGEXP` operators and `REGEXP_CONTAINS`/`REGEXP_EXTRACT`
> functions. It implements the RE2 syntax subset, which is close enough to
> BigQuery's RE2 for practical use. No external dependency needed.

---

## Built-in functions

### String

| Function                   | Return  | Notes                                    |
|:---------------------------|:--------|:-----------------------------------------|
| `LOWER(s)`                 | string  |                                          |
| `UPPER(s)`                 | string  |                                          |
| `LENGTH(s)`                | int     | Unicode codepoint count                  |
| `SUBSTR(s, pos[, len])`    | string  | 1-based; negative pos counts from end    |
| `STARTS_WITH(s, prefix)`   | bool    |                                          |
| `ENDS_WITH(s, suffix)`     | bool    |                                          |
| `CONTAINS_SUBSTR(s, needle)` | bool  |                                          |
| `TRIM(s[, chars])`         | string  | Both ends; chars default = whitespace    |
| `LTRIM(s[, chars])`        | string  | Left end                                 |
| `RTRIM(s[, chars])`        | string  | Right end                                |
| `REPLACE(s, from, to)`     | string  | Literal substring replace, all occurrences |
| `SPLIT(s, delim)`          | list    |                                          |
| `CONCAT(s, …)`             | string  | ≥2 args                                  |
| `REGEXP_CONTAINS(s, re)`   | bool    | Go `regexp` (RE2-compatible subset)      |
| `REGEXP_EXTRACT(s, re)`    | string  | First capture group or null              |
| `TO_STRING(x)`             | string  | Explicit cast to string                  |

### Numeric / arithmetic

| Function           | Return  | Notes                                        |
|:-------------------|:--------|:---------------------------------------------|
| `ABS(x)`           | number  | Same kind as input (int→int, number→number)  |
| `CEIL(x)`          | int     |                                              |
| `FLOOR(x)`         | int     |                                              |
| `ROUND(x[, n])`    | number  | n decimal places, default 0                  |
| `MOD(x, y)`        | int     | Integer modulo; null on div-by-zero          |
| `SQRT(x)`          | number  |                                              |
| `POW(x, y)`        | number  |                                              |
| `GREATEST(x, …)`   | any     | Max of args (null-skipping)                  |
| `LEAST(x, …)`      | any     | Min of args (null-skipping)                  |
| `COALESCE(x, …)`   | any     | First non-null arg                           |

### Set / list

| Function                   | Return  | Notes                               |
|:---------------------------|:--------|:------------------------------------|
| `ARRAY_LENGTH(list)`       | int     |                                     |
| `ARRAY_CONTAINS(list, e)`  | bool    | Same semantics as `list >= scalar`  |
| `ARRAY_CONCAT(list, …)`    | list    | Concatenate; preserves duplicates   |
| `DISTINCT(list)`           | list    | Remove duplicates, preserve order   |
| `ARRAY_TO_STRING(list, sep)` | string | Join                               |

### Date / time

| Function        | Return | Notes                                 |
|:----------------|:-------|:--------------------------------------|
| `TODAY()`       | date   | Current local date                    |
| `YEAR(d)`       | int    | Works on date and datetime            |
| `MONTH(d)`      | int    | 1–12                                  |
| `DAY(d)`        | int    | Day of month                          |
| `DATE_DIFF(a,b)`| int    | Days: a − b (positive when a > b)     |

---

## Implementation files

| File                         | Change |
|:-----------------------------|:-------|
| `internal/ast.go`            | Add `FuncExpr`; rename/add `BinOp` constants |
| `internal/parse.go`          | Dispatch `FuncExpr` in `parsePrimary`; add infix keyword operators in `parseComparison` |
| `internal/eval_func.go`      | New file: `FuncExpr.Eval` dispatch + all function implementations |
| `internal/eval.go`           | Add `BinIntersect` / `BinUnion` to `BinExpr.Eval`; add like/regexp to `compare` |
| `internal/eval_expr_test.go` | Tests for new operators |
| `internal/parse_test.go`     | Tests for function call parsing and new operator parsing |
| `internal/eval_func_test.go` | New file: function eval tests |

---

## Tests to validate

1. **Parse round-trips** — each new function name and operator parses without error.
2. **`<=>` intersection** — `["a","b","c"] <=> ["b","c","d"]` → `["b","c"]`.
3. **`>=<` union** — `["a","b"] >=< ["b","c"]` → `["a","b","c"]`.
4. **LIKE / ILIKE / NOT LIKE** — pattern `%foo%` matches / doesn't match with correct case sensitivity.
5. **REGEXP / NOT REGEXP** — RE2 match and negation, including raw string literals `r'^\d+'`.
6. **String functions** — LOWER, UPPER, LENGTH, SUBSTR, STARTS_WITH, ENDS_WITH, CONTAINS_SUBSTR, TRIM, REPLACE, SPLIT, CONCAT, REGEXP_CONTAINS, REGEXP_EXTRACT, TO_STRING.
7. **Numeric functions** — ABS, CEIL, FLOOR, ROUND, MOD, SQRT, POW, GREATEST, LEAST, COALESCE.
8. **List functions** — ARRAY_LENGTH, ARRAY_CONTAINS, ARRAY_CONCAT, DISTINCT, ARRAY_TO_STRING.
9. **Date functions** — TODAY returns current date; YEAR/MONTH/DAY extract components; DATE_DIFF returns signed day count.
10. **Null propagation** — functions receiving null args return null (except COALESCE which skips nulls).
11. **Arity errors** — wrong number of args returns null (not a panic).
12. **FROM-less select smoke test** — `fm 'select UPPER("hello"), ARRAY_LENGTH([1,2,3])'` produces `HELLO  3`.
13. **WHERE clause with functions** — `select title from *.md where STARTS_WITH(title, "2024")`.
14. **`<=>` / `>=<` in WHERE** — `select * from *.md where ARRAY_LENGTH(tags <=> ["go","python"]) > 0`.
