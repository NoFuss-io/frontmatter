baseline = 36b91a4
target  = TARGET_260606_pluggable_engines.md

# Implementation plan — pluggable store API

Phased extraction of the markdown I/O layer into the `store.Store` / `store.Format`
interfaces. Each phase is its own PR and leaves the test suite green. Later
phases never depend on phases not yet merged.

---

## Phase A — Define the `store/` package

Create `store/store.go` with the three public types. No logic, no imports from
`internal/`.

- [ ] Create `store/store.go`:
  - `type Store interface { Enumerate / Read / Write / Label }`
  - `type Format interface { Read / Write }` (no `Label`)
  - `type FileStore struct { Fmt Format }` with method stubs (bodies in Phase B)
  - `type EnumOptions struct { IncludeHidden bool }`
- [ ] Verify the package compiles in isolation:
      `go build github.com/nofuss-io/frontmatter/store`
- [ ] Commit: `feat: add store.Store / store.Format / store.FileStore interfaces`.

---

## Phase B — Implement `FileStore`

Fill in `FileStore.Enumerate` with the glob expansion and hidden-file filter
logic extracted from `internal/`. Nothing in `internal/` changes yet.

- [ ] Move `ExpandGlobs` from `internal/file.go` into `store/store.go` as an
  unexported helper used by `FileStore.Enumerate`.
- [ ] Move `filterHidden` from `internal/exec.go` into the same helper,
  gated on `opts.IncludeHidden`.
- [ ] Implement `FileStore.Read`, `FileStore.Write` as delegation to `fs.Fmt`.
- [ ] Implement `FileStore.Label` as `return key`.
- [ ] Unit-test `FileStore.Enumerate` in `store/store_test.go`: glob hits,
  hidden-file filtering (on and off), bare path, nonexistent path error.
- [ ] `go build ./...` and `just test` green.
- [ ] Commit: `feat: implement FileStore with shared glob expansion`.

---

## Phase C — Extract markdown store

Create `store/markdown/` implementing `store.Format`. No changes to `internal/`
yet — this is additive only.

- [ ] Create `store/markdown/markdown.go`:
  - `type Format struct { bodies sync.Map }` — sidecar for body round-trip.
  - `func New() store.Store { return store.FileStore{Fmt: &Format{}} }`
  - `Read(path string)`: call current `internal.ReadDocument` logic (copy or
    call — either works at this stage); store body in `bodies`; return
    `doc.FrontMatter`.
  - `Write(path string, fields map[string]any)`: load body from `bodies`;
    reconstruct `Document{FrontMatter: fields, Body: body}`; write to disk.
  - Keep `Document` struct here; it is no longer needed anywhere else.
- [ ] Unit-test in `store/markdown/markdown_test.go`:
  - `Read` on a file with a valid `---` block returns expected fields.
  - `Read` on a file without a `---` block returns empty map, body preserved.
  - `Read` → mutate fields → `Write` → `Read` round-trip: fields updated,
    body unchanged, file on disk matches.
- [ ] `just test` green.
- [ ] Commit: `feat: add store/markdown implementing store.Format`.

---

## Phase D — Wire `Store` into the executor

Update `internal/exec.go` to delegate to `store.Store`. All existing tests
pass because the CLI still supplies the markdown store.

- [ ] Add `import "github.com/nofuss-io/frontmatter/store"` to `internal/exec.go`.
- [ ] Add `Store store.Store` field to `ExecOptions`.
- [ ] In `Run`, if `opts.Store == nil` use `markdown.New()` as the default.
  (Requires importing `store/markdown` into `internal/exec.go`. If that
  feels like a layering violation, push the nil-default into `cmd/fm` and
  make a nil Store an explicit error in `Run` instead — either is valid.)
- [ ] Replace `expandPlan`: call `s.Enumerate(stmt.Patterns(), store.EnumOptions{…})`
  instead of `ExpandGlobs` + `filterHidden`.
- [ ] Replace `ReadDocument(path)` in the item loop with `s.Read(key)`.
- [ ] Replace `Write(path, doc)` in write-back with `s.Write(key, fm)`.
- [ ] Replace bare-path label in output with `s.Label(key)`.
- [ ] `ExpandGlobs` and `filterHidden` in `internal/` are now dead — leave
  them for Phase E removal.
- [ ] `just test` and `go test ./...` green.
- [ ] Commit: `refactor: exec delegates I/O to store.Store`.

---

## Phase E — Rename `Globs()` to `Patterns()`

Cosmetic rename on the `Query` interface; no behavioral change.

- [ ] In `internal/ast.go`, rename `q.Globs() []string` → `q.Patterns() []string`
  on the `query` struct and the `Query` interface.
- [ ] Update all call sites: `internal/exec.go`, `internal/result.go`, any
  tests that call `.Globs()` directly.
- [ ] `just test` green.
- [ ] Commit: `refactor: rename Query.Globs() → Query.Patterns()`.

---

## Phase F — Remove orphaned internal symbols

Delete the markdown-specific code that now lives in `store/markdown/` and the
glob code that now lives in `store/`.

- [ ] Delete `ExpandGlobs` and any remaining glob helpers from `internal/file.go`.
- [ ] Delete `filterHidden` from `internal/exec.go`.
- [ ] Delete `ReadDocument`, `Write`, `Document` from `internal/document.go`
  (delete the file if it is now empty).
- [ ] Fix any compilation errors; `just lint && just test` green.
- [ ] Commit: `chore: remove orphaned internal markdown I/O after store extraction`.

---

## Phase G — Update public re-exports (`frontmatter.go`)

- [ ] Remove re-exports: `ExpandGlobs`, `ReadDocument`, `Write`, `Document`.
- [ ] Add re-exports: `store.Store`, `store.Format`, `store.FileStore`, `store.EnumOptions`.
- [ ] Add a re-export or doc pointer for `store/markdown.New` so library
  consumers can construct the default store without a separate import.
- [ ] `go build ./...` clean; `just test` green.
- [ ] Commit: `feat: update public API — expose store types, remove I/O re-exports`.

---

## Phase H — Docs and baseline update

- [ ] Update `docs/manual.md` if any flag or behavioural descriptions changed.
- [ ] Update `README.md` / `AGENTS.md` if they reference removed symbols.
- [ ] Update `architecture/BASELINE.md`:
  - Package table: add `store/`, `store/markdown/`; update `internal/` description.
  - Remove mentions of `ExpandGlobs`, `ReadDocument`, `Write` as re-exported symbols.
  - Add `Store` / `Format` / `FileStore` interface descriptions and the new
    dependency graph.
  - Bump baseline SHA.
- [ ] Commit: `docs: update baseline and manual for store API`.

---

## Risk notes

| Risk | Mitigation |
|:-----|:-----------|
| Body round-trip correctness | Unit-test the full Read → Write → Read cycle; run the e2e suite |
| `sync.Map` sidecar lifetime | `Format` is created once per `Run` call via `markdown.New()`; the sidecar is GC'd when `Run` returns |
| `internal/` importing `store/markdown` for nil-default | Push nil-to-default into `cmd/fm` if preferred; make nil Store an error in `Run` |
| Breaking public API | Changelog entry; BREAKING prefix in commit subject for Phases F and G |
