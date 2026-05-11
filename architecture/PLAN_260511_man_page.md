# Plan: Man page + Homebrew distribution

## Phase 1 — Man page

### 1a — Enrich command descriptions

Edit `cli/main.go`:

1. Add `Long` to the root `fm` command — one-paragraph description of the tool and its
   Obsidian/knowledge-base use case.
2. Add `Long` to `selectCmd()` — describe output format, field syntax, sort/limit.
3. Add `Long` to `updateCmd()` — describe all assignment forms (set, cast, +=, -=).
4. Add `Long` to `alterCmd()` — describe typed vs untyped drop behaviour.

### 1b — Add `gen-man` and `install-man` subcommands

Edit `cli/main.go`:

5. Import `github.com/spf13/cobra/doc`.
6. Add `genManCmd()` — hidden subcommand, one optional arg for output directory
   (default `.`). Calls `doc.GenMan(root, header, file)` with section 1 and
   source string `fm <VERSION>`.
7. Add `installManCmd()` — hidden subcommand with no args. Finds the first writable
   `man1/` directory via `man --manpath`, creates it if needed, writes `fm.1` there.
8. Register both on the root command.

### 1c — Update dependencies

9. Run `go get github.com/spf13/cobra/doc` and `just vendor` to update `go.mod`,
   `go.sum`, and `vendor/`.

### 1d — Just recipes and generated file

10. Add private recipes to `justfile`:
    ```
    [private]
    _gen-man: build
        ./fm gen-man docs/

    [private]
    _install-man: _gen-man
        ./fm install-man
    ```
11. Run `just _gen-man` to produce `docs/man/fm.1`.
12. Commit: `cli/main.go`, `go.mod`, `go.sum`, `vendor/`, `docs/man/fm.1`, `justfile`.

---

## Phase 2 — Homebrew distribution

13. Create `scripts/fm.rb.tmpl` — Homebrew formula template with `{{VERSION}}` and
    `{{SHA256}}` placeholders. Formula body:
    - `url` pointing to the GitHub release tarball.
    - `depends_on` Go.
    - `install` block: `go build`, `man1.install "docs/man/fm.1"`,
      completions for bash/zsh/fish via `fm completion <shell>`.
    - `test` block: `fm --version`.
14. Create `scripts/make-formula.sh`:
    - Usage: `make-formula.sh <tag>` (e.g. `v0.2.0`).
    - Derives tarball URL `https://github.com/backlin/frontmatter/archive/refs/tags/<tag>.tar.gz`.
    - Downloads tarball, computes SHA256, substitutes into template, prints result to
      stdout. Maintainer copies output to `Formula/fm.rb` in the `homebrew-backlin` tap
      repo.
15. Add private just recipe:
    ```
    [private]
    _formula tag:
        scripts/make-formula.sh {{tag}}
    ```
16. Commit: `scripts/fm.rb.tmpl`, `scripts/make-formula.sh`, `justfile`.
