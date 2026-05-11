# Target: Man page + Homebrew distribution

Add a `man fm` entry generated from the Cobra command tree, and a Homebrew tap formula
so users can install the binary, man page, and shell completions in one step.

---

## Phase 1 — Man page

Use `github.com/spf13/cobra/doc` to auto-generate the man page from the command
definitions. Generation is exposed as a hidden subcommand `fm gen-man [dir]` so no
separate build tool or repo clone is needed — users who install via `go install` can
run `fm gen-man` then `fm install-man` to also get the man page. The generated
`docs/man/fm.1` is committed to the repo as a convenience.

### Changes

**`cli/main.go`**
- Add `Long` descriptions to the root command and all three subcommands.
- Add hidden subcommand `gen-man [dir]` (default dir `.`) that generates `fm.1`.
- Add hidden subcommand `install-man` that places `fm.1` under the first writable
  directory on `$(man --manpath)`.

**`go.mod` / `vendor/`**
- Add `github.com/spf13/cobra/doc` (sub-package of already-vendored cobra module).

**`docs/man/fm.1`**
- Generated man page committed to the repo.

**`justfile`** — maintainer recipes (private, prefixed `_`):
- `_gen-man`: builds binary and runs `./fm gen-man docs/man/` to regenerate `docs/man/fm.1`.
- `_install-man`: calls `_gen-man` then runs `fm install-man`.

---

## Phase 2 — Homebrew distribution

A Homebrew tap (`github.com/backlin/homebrew-backlin`, separate repo) lets users
install with `brew install backlin/backlin/fm`. The formula installs the binary, man
page, and shell completions.

### Changes

**`scripts/make-formula.sh`** (new, maintainer-only)
- Takes a git tag as argument, computes the source tarball URL and SHA256, and renders
  `scripts/fm.rb.tmpl` into a ready-to-commit `Formula/fm.rb`.
- Output is printed to stdout; maintainer copies it into the tap repo at
  `Formula/fm.rb`.

**`scripts/fm.rb.tmpl`** (new)
- Homebrew formula template with `{{VERSION}}` and `{{SHA256}}` placeholders.
- Installs binary via `go build`, installs `docs/man/fm.1` as a man page, generates and
  installs shell completions via `fm completion <shell>`.

**`justfile`** — maintainer recipe:
- `_formula tag`: runs `scripts/make-formula.sh {{tag}}`.

---

## Out of scope

- CI/CD release automation (GitHub Actions for tagging and formula update).
- Submission to homebrew-core.
- Section 5 or section 7 man pages.
