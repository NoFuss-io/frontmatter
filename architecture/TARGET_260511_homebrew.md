# Target: Homebrew distribution

Homebrew tap formula
so users can install the binary, man page, and shell completions in one step.

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
  installs shell completions via `generate_completions_from_executable(bin/"fm", "completion")`.

**`justfile`** — maintainer recipe:
- `_formula tag`: runs `scripts/make-formula.sh {{tag}}`.

---

## Out of scope

- CI/CD release automation (GitHub Actions for tagging and formula update).
- Submission to homebrew-core.
