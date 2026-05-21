baseline = 216bc548

# Target state — first-release tidy-up

Four scoped cleanups to land before v0.1.0:

## 1. Whole-file parser

Drop streaming `bufio.Reader` style throughout `lib/parse.go`. Replace with a
single in-memory cursor (`pos int` over `[]byte` or `string`).

- `ParseProgram(r io.Reader)` reads `io.ReadAll(r)` first, then dispatches on a
  cursor type.
- Internal helpers (`peekRune`, `readIdent`, `readQuotedIdent`, `readGlobs`,
  `readFieldList`, `readExprList`, `readAssignList`, `readRenamePairs`,
  `readSortTermList`, `parseOneQuery`, all `parseOrExpr`/`parseAndExpr`/
  `parseComparison`/`parseArith`/`parseTerm`/`parseFactor`/`parsePrimary`,
  `LitExpr.parseString`/`parseNumber`/`parseKeyword`, `Field.parse`,
  `Assign.parse`, `SortTerm.parse`, `RenamePair.parse`, `expectKeyword`,
  `atStopKeyword`, `skipWS`, `peekKeyword`, `parseOptionalWhere`, etc.) all
  operate on the cursor.
- Public single-node entrypoints (`Field.Parse`, `Assign.Parse`,
  `SortTerm.Parse`, `RenamePair.Parse`, `LitExpr.Parse`, `ParseExpr`) keep their
  `io.Reader` signatures but internally slurp + delegate.
- Behaviour is unchanged; existing parser tests must still pass.

## 2. List type — strings only

Lists are list-of-string. No `list:<elemtype>` syntax. Removes a whole
combinatorial branch from the type system without losing the YAML round-trip.

- `Field.ElemType` removed from `lib/ast.go`.
- Parser no longer accepts `list:<elem>`; a colon after `list` is a parse error.
- `CastField` collapses to `Cast(v, TypeList)` with the additional rule that
  each element is cast to `TypeString`.
- Tests for `list:int`, `list:date`, etc. either drop or convert to plain `list`.
- `docs/Manual.md` and any README/tutorial examples updated.

## 3. `+=` to empty/null list creates `null` first element — fix

Repro: a frontmatter field with value `null` (e.g. produced by cast-only form
`update file set tags:list`) followed by `set tags += "foo"` yields
`tags: [null, "foo"]` instead of `tags: ["foo"]`.

Fix in `applyListAdd` (`lib/exec.go`): if the current value is `nil` (Go
`nil`, i.e. YAML null), treat as empty list rather than wrapping the `nil` into
`[]any{nil}`. Regression test in `lib/exec_test.go`.

Same fix applies to `applyListSub` — subtracting from a `nil` field should be a
no-op, never produce a `[nil]` then filter.

## 4. Rename `Void` → `Null`

`Value.Void` is the absence sentinel. Conceptually it's the same thing YAML
calls `null`, the `null` literal already lowers to it, and we have a
`LitNull` kind. "Void" is jargon imported from C-family return types; "null"
matches the data model.

- `Value.Void` field → `Value.Null` in `lib/exec.go` (and on the type
  declaration).
- All `v.Void`, `Value{Void: true}` updated.
- Doc comments that say "void" should now say "null". `Value.String()` returns
  `"null"` instead of `"void"`.
- `TypeAny` semantics unchanged — only the field name moves.

## Out of scope

- Window/aggregate clauses (see `FEATURE_WINDOWS.md`).
- Homebrew distribution (see `TARGET_260511_homebrew.md`).
- Any new query forms, operators, or types.
- Function calls in expressions.
- CLI flag changes.
