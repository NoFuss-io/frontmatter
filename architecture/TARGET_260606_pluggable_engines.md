baseline = 36b91a4

# Target state — pluggable store API

Decouple the SQL DSL (parsing + evaluation) from the markdown file I/O layer so
that third-party code — and future first-party backends — can implement a
different *store* that reads and writes a different kind of record corpus:
image EXIF data, Jira issues, SQLite rows, etc.

The DSL syntax, parser, evaluator, and output layer do not change. Only the
data-source boundary is extracted into a formal interface.

---

## Mental model

Today the executor in `internal/exec.go` directly calls `ExpandGlobs`,
`ReadDocument`, and `Write`. These three operations are the only things that
know about Markdown files. Everything else — parsing the query, evaluating
expressions, filtering WHERE, projecting columns, sorting, rendering — is
already format-agnostic. The refactor makes that separation explicit by
extracting those three operations behind a `Store` interface.

```
Before:
  exec.go → ExpandGlobs / ReadDocument / Write   (hardwired markdown)

After:
  exec.go → Store.Enumerate / Store.Read / Store.Write   (interface)
       ↑
  store.FileStore{markdown.Format{}}   (current behaviour, reuses shared glob expansion)
  store.FileStore{image.Format{}}      (future — EXIF, same glob expansion)
  jira.Store{}                         (future — implements Store directly)
```

The key insight is that glob expansion is **not** specific to Markdown: any
file-based store (image, audio, video) would expand globs the same way. Only
API-based stores (Jira, Notion, GitHub) have their own enumeration logic.
Two interfaces capture this split cleanly.

---

## New public package: `store/`

Path: `github.com/nofuss-io/frontmatter/store`

### `Store` — what the executor sees

```go
// Store is the pluggable data-source backend. The executor calls it to
// enumerate items, read their fields, write mutations back, and label them
// in output tables.
type Store interface {
    // Enumerate resolves FROM-clause pattern tokens into opaque item keys.
    // File stores receive glob strings; API stores receive domain-specific
    // identifiers (project keys, JQL, URLs, …). The store interprets them.
    Enumerate(patterns []string, opts EnumOptions) ([]string, error)

    // Read returns the field map for one item. The key is a value previously
    // returned by Enumerate.
    Read(key string) (map[string]any, error)

    // Write persists a mutated field map. Only called when a mutation
    // statement succeeded and DryRun is false.
    Write(key string, fields map[string]any) error

    // Label returns a human-readable display name for the key, used as the
    // row identifier in output tables (the "filename" column).
    Label(key string) string
}

// EnumOptions carries per-enumeration settings that the store may consult.
type EnumOptions struct {
    // IncludeHidden requests that items whose names begin with '.' not be
    // filtered out. File stores apply this to basenames; API stores ignore it.
    IncludeHidden bool
}
```

### `Format` — what file-format authors implement

```go
// Format handles reading and writing a single file. Glob expansion and hidden-
// file filtering are provided by FileStore, so Format authors only need to
// care about the file contents.
type Format interface {
    Read(path string) (map[string]any, error)
    Write(path string, fields map[string]any) error
}
```

`Label` is intentionally absent from `Format`. For any file-based store the
label is always the path — there is no reason to make each format repeat that
trivially. API stores that need a non-path label (e.g. a Jira issue key like
`PROJ-123`) implement `Store` directly and supply their own `Label`.

### `FileStore` — shared base for file-based stores

```go
// FileStore is a concrete Store for any file-based format. It implements
// Enumerate (glob expansion + hidden-file filter) once, and delegates
// Read/Write to the supplied Format. Label returns the path unchanged.
type FileStore struct {
    Fmt Format
}

func (fs FileStore) Enumerate(patterns []string, opts EnumOptions) ([]string, error) { … }
func (fs FileStore) Read(key string) (map[string]any, error)                          { return fs.Fmt.Read(key) }
func (fs FileStore) Write(key string, fields map[string]any) error                    { return fs.Fmt.Write(key, fields) }
func (fs FileStore) Label(key string) string                                          { return key }
```

`FileStore` contains the glob expansion and hidden-basename filtering logic
currently in `internal/file.go` (`ExpandGlobs`) and `internal/exec.go`
(`filterHidden`). It is implemented once and shared by all file-based stores.

This package has no imports from `internal/`. Third-party store authors
import only `store/` to implement either `Format` (file-based) or `Store`
(API-based).

---

## New public package: `store/markdown/`

Path: `github.com/nofuss-io/frontmatter/store/markdown`

Implements `store.Format` for Markdown files with YAML frontmatter. Contains
the logic currently in `internal/document.go` (`ReadDocument`, `Write`,
`Document`).

```go
package markdown

// Format implements store.Format for Markdown files with YAML frontmatter.
type Format struct{}

func New() store.Store { return store.FileStore{Fmt: Format{}} }
```

| Method | Current home | Behavior |
|:-------|:-------------|:---------|
| `Read` | `internal/document.go:ReadDocument` | Parse `---`-fenced YAML; missing fence → empty map. Stash body for round-trip. |
| `Write` | `internal/document.go:Write` | Reconstruct `---`-fenced file from stashed body + mutated fields. |

`Document` moves here (it is only needed to pair frontmatter with its body for
the round-trip write; no other code needs it). The body sidecar — required
because `Read` returns only `map[string]any` but `Write` must preserve the
non-frontmatter body — is stored in a `sync.Map` keyed by path on the `Format`
struct. A `Format` instance is created once per `Run` call so the sidecar is
short-lived.

---

## Changes to `internal/exec.go`

`ExecOptions` gains one field:

```go
// Store selects the data-source backend. Nil defaults to
// store/markdown.New() — preserving the current behaviour.
Store store.Store
```

Inside `Run`:
- `expandPlan` calls `s.Enumerate(stmt.Patterns(), store.EnumOptions{IncludeHidden})` instead of `ExpandGlobs` + `filterHidden`.
- The item loop calls `s.Read(key)` instead of `ReadDocument`.
- Write-back calls `s.Write(key, fm)` instead of `Write(path, doc)`.
- The "filename" column uses `s.Label(key)` instead of the bare path.

`expandPlan` shape is unchanged — it still returns `(map[int][]string, []string, error)`.

---

## Changes to `internal/ast.go`

`Query.Globs() []string` is renamed to `Query.Patterns() []string`.

"Globs" is a filesystem-ism. "Patterns" is neutral: for file stores the
patterns are glob strings; for a Jira store they might be project keys. No
change to values or semantics — only the method name.

`Query.Eval(fm FrontMatter) (*TableRow, error)` is unchanged. The executor
reads a `map[string]any` from `store.Read`, which is directly assignable to
`FrontMatter` (an alias for `map[string]any`), so the evaluator needs no
update.

---

## Changes to `frontmatter.go` (public re-exports)

| Symbol | Change |
|:-------|:-------|
| `ExpandGlobs` | **removed** — logic moved to `store.FileStore` |
| `ReadDocument` | **removed** — moved to `store/markdown` |
| `Write` | **removed** — moved to `store/markdown` |
| `Document` | **removed** — moved to `store/markdown` |
| `store.Store` | **added** re-export |
| `store.Format` | **added** re-export |
| `store.FileStore` | **added** re-export |
| `store.EnumOptions` | **added** re-export |
| `Query.Globs()` | **renamed** to `Query.Patterns()` |

`FrontMatter` remains as a public type alias for `map[string]any`.

These are breaking changes on the public library surface. `fm` is still
alpha-stage software; the breakage is acceptable and documented in the
changelog.

---

## Package dependency graph (after)

```
store/                   (Store + Format interfaces + FileStore; no internal deps)
    ↑
internal/                (imports store/ for Store interface)
store/markdown/          (imports store/ + internal/)
frontmatter.go           (re-exports from internal/ + store/)
cmd/fm/                  (imports frontmatter + store/markdown)
```

No cycles. Third-party store authors import only `store/` and nothing from
`internal/`.

---

## Implementing a new store

**File-based** (e.g. image EXIF): implement `store.Format` — two methods.

```go
store.FileStore{Fmt: image.Format{}}
// glob expansion, hidden filter, and Label are provided for free
```

**API-based** (e.g. Jira): implement `store.Store` — four methods.

```go
jira.Store{Client: …}
// Enumerate calls the Jira search API; Label returns the issue key
```

---

## What does NOT change

- Parser (`internal/parse.go`) — untouched
- Evaluator (`internal/eval.go`, `eval_query.go`) — untouched
- Result / format / table packages — untouched
- CLI flags — no new `--store` flag; markdown is the compiled-in default
- DSL syntax — `FROM` clause still takes opaque pattern strings; stores interpret them
- E2e test behaviour — all existing golden tests continue to pass

---

## Illustrative future stores (out of scope for this target)

| Store | Implements | `FROM` tokens | Notes |
|:------|:-----------|:--------------|:------|
| `store/image` | `Format` | file glob (`photos/*.jpg`) | EXIF read/write via `dsoprea/go-exif`; reuses `FileStore` |
| `jira` | `Store` | project key or JQL | Enumerate via Jira search API; Label returns issue key |
| `sqlite` | `Store` | `table_name` | Enumerate via `SELECT rowid`; no glob involved |

A `--store` CLI flag (or a `fm-jira` sibling binary) is a natural follow-on
once a second store exists.
