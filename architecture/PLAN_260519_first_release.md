baseline = 216bc548

# Implementation plan — first-release tidy-up

Four independent changes. Each is its own commit so they can be reviewed and
reverted in isolation.

## Phase 1 — Fix `+=` to null/empty list bug

Smallest change, lands first so subsequent refactors don't muddy the bisect.

- [x] Add regression test in `lib/exec_test.go`: field set to YAML `null`,
      then `+=` scalar; expect single-element list, no leading `null`.
- [x] Add regression test: same scenario for `-=` (no-op on null field).
- [x] Update `applyListAdd` in `lib/exec.go`: when `cur == nil` (the `any`
      value, not `!ok`), start with empty list instead of wrapping nil.
- [x] Update `applyListSub`: when `cur == nil`, return without touching map.
- [x] `just test` green.
- [x] Commit: `fix: drop null leading element when += to null-valued field`.

## Phase 2 — Rename `Void` → `Null`

Mechanical rename. Done before parser rewrite so the rewrite sees the new name.

- [x] `lib/exec.go`: `Value.Void bool` → `Value.Null bool`. Update field
      initialisers, comparisons, and the `Value.String` "void" → "null"
      branch.
- [x] `lib/exec.go` doc comments mentioning void → null.
- [x] `lib/format.go`: `FormatValue` and `compareValues` void checks.
- [x] `lib/exec_test.go`, `lib/exec_cast_test.go`, `lib/parse_test.go`:
      replace `Void` field references and any string assertions of
      `"void(…)"` with `"null(…)"`.
- [x] `docs/Manual.md`: any mention of "void" semantics → "null".
- [x] `just test` green.
- [x] Commit: `refactor: rename Value.Void to Value.Null`.

## Phase 3 — Drop `list:<elem>` syntax

Type-system simplification. Independent of parser rewrite.

- [x] `lib/ast.go`: remove `Field.ElemType *FieldType`.
- [x] `lib/parse.go` `(*Field).parse`: drop the `if ft == TypeList { … colon
      and elem type … }` block. A `:` after `list` now falls through to
      whatever caller expects next; if followed by an ident it should error
      with a clear message.
- [x] `lib/exec.go` `CastField`: replace list-element-typed branch with a
      flat list cast where every element is forced to `TypeString` (use the
      existing `castToString` path; failures surface per element).
- [x] `lib/exec.go` `Cast(target=TypeList)` scalar-wrap path: wrap as a
      `TypeString`-coerced element so list contents are uniformly strings.
- [x] `lib/parse_test.go`: remove `list:int` / `list:date` / `list:string`
      cases. Keep `list` cases. Add a test asserting `list:string` (and any
      other `list:<x>`) now produces a parse error.
- [x] `lib/exec_test.go` / `lib/exec_cast_test.go`: replace any
      `ElemType`-bearing fixtures with plain list-of-string equivalents.
- [x] `docs/Manual.md`: drop `list:<elem>` documentation, note lists are
      strings.
- [x] `README.md`: tidy any list-typed examples.
- [x] `docs/tutorial.md` and `docs/tutorial_*.sh|sql`: check no `list:<elem>`
      uses; convert if any.
- [x] `just test` green.
- [x] Commit: `refactor: lists are list-of-string, drop list:<elem> syntax`.

## Phase 4 — Whole-file parser

Largest change, lands last. Behaviour-preserving rewrite of `lib/parse.go`.

- [x] Introduce a private cursor type in `lib/parse.go`, e.g.

      ```go
      type cursor struct {
          src []byte
          pos int
      }
      func newCursor(b []byte) *cursor
      func (c *cursor) peekRune() (rune, int)   // size 0 at EOF
      func (c *cursor) readRune() (rune, int)
      func (c *cursor) peek(n int) []byte
      func (c *cursor) advance(n int)
      func (c *cursor) eof() bool
      ```

- [x] Rewrite `skipWS`, `peekKeyword`, `expectKeyword`, `atStopKeyword`,
      `consumeBytes` (renamed `advance`), `readIdent`, `readQuotedIdent`,
      `readFieldName`, `readIntLit`, `readHexChars` against the cursor.
- [x] Rewrite clause helpers (`readGlobs`, `readFieldList`, `readExprList`,
      `readSortTermList`, `readAssignList`, `readRenamePairs`,
      `parseOptionalWhere`) against the cursor.
- [x] Rewrite per-query parsers (`SelectQuery.parse`, `UpdateQuery.parse`,
      `AlterQuery.parse`) and `parseOneQuery`, `ParseProgram`, `ParseQuery`,
      `skipSeparators`.
- [x] Rewrite expression parsers (`parseOrExpr`/`AndExpr`/`NotExpr`/
      `Comparison`/`Arith`/`Term`/`Factor`/`Primary`, `LitExpr.parse`/
      `parseString`/`parseNumber`/`parseKeyword`).
- [x] Rewrite per-node `Parse(io.Reader)` entrypoints to slurp via
      `io.ReadAll` then delegate to the cursor variant.
- [x] Remove `bufio` import once all references are gone.
- [x] `just test` green — the existing test suite is the behavioural
      contract; no test changes expected beyond mechanical signature
      tweaks if helpers are renamed.
- [x] Commit: `refactor: parse from in-memory cursor instead of bufio.Reader`.

## Phase 5 — Baseline refresh

- [x] Update `architecture/BASELINE.md`:
      - First-line SHA bumped to the post-merge commit.
      - Type table: drop `list:<elem>`.
      - `Field` row: drop `ElemType`.
      - `Value` block: `Void` → `Null`.
      - Parser section: "Recursive-descent over an in-memory byte cursor".
      - Any other rot from the four changes above.
- [x] Commit: `docs: refresh architecture baseline for first release`.
