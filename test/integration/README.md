# Integration tests

Table-driven end-to-end tests against the compiled `fm` binary. Each
subdirectory under `cases/` is one test case.

## Layout

```
cases/<case-name>/
├── input/                  # (optional) .md files copied to a fresh tmpdir
├── cmd                     # shell script run via `sh -c` from the tmpdir
├── expected                # golden stdout
├── expected_stderr         # (optional) golden stderr
├── expected_files/         # (optional) post-run file contents
│   └── note.md
└── expect_failure          # (optional, empty file) tolerate non-zero exit
```

The runner builds `fm` once, prepends its directory to `PATH`, then for each
case:

1. Copies `input/` into a fresh `t.TempDir()`.
2. Executes `cmd` with `sh -c` in that tmpdir.
3. Diffs captured stdout against `expected` (and stderr against
   `expected_stderr` if present).
4. If `expected_files/` exists, diffs every file under it against the same
   relative path in the tmpdir (catches mutations).

## Running

```sh
go test ./test/integration/...
```

## Updating goldens

```sh
go test ./test/integration/... -update
```

Writes `expected`, `expected_stderr`, and `expected_files/*` from the current
run output. Review the diff before committing.

## Adding a case

```sh
mkdir -p test/integration/cases/my-case/input
cat > test/integration/cases/my-case/input/note.md <<'EOF'
---
tag: a
---
EOF
cat > test/integration/cases/my-case/cmd <<'EOF'
fm 'select tag from *.md'
EOF
go test ./test/integration/... -update -run TestCases/my-case
```
