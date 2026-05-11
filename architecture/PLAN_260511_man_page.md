# Plan: Man page + Homebrew distribution

## Phase 1 — Man page

### 1a — Enrich command descriptions

Edit `cli/main.go`:

- [x] Add `Long` to the root `fm` command — one-paragraph description of the tool and its
      Obsidian/knowledge-base use case.
- [x] Add `Long` to `selectCmd()` — describe output format, field syntax, sort/limit.
- [x] Add `Long` to `updateCmd()` — describe all assignment forms (set, cast, +=, -=).
- [x] Add `Long` to `alterCmd()` — describe typed vs untyped drop behaviour.

### 1b — Add `gen-man` and `install-man` subcommands

Edit `cli/main.go`:

- [x] Import `github.com/spf13/cobra/doc`.
- [x] Add `genManCmd()` — hidden subcommand, one optional arg for output directory
      (default `.`). Calls `doc.GenMan(root, header, file)` with section 1 and
      source string `fm <VERSION>`.
- [x] Add `installManCmd()` — hidden subcommand with no args. Finds the first writable
      `man1/` directory via `man --manpath`, creates it if needed, writes `fm.1` there.
- [x] Register both on the root command.

### 1c — Update dependencies

- [x] Run `go get github.com/spf13/cobra/doc` and `just vendor` to update `go.mod`,
      `go.sum`, and `vendor/`.

### 1d — Just recipes and generated file

- [x] Add private recipes to `justfile`:
      ```
      [private]
      _gen-man: build
          mkdir -p docs/man
          ./fm gen-man docs/man/

      [private]
      _install-man: _gen-man
          ./fm install-man
      ```
- [x] Run `just _gen-man` to produce `docs/man/fm.1`.
- [x] Commit: `cli/main.go`, `go.mod`, `go.sum`, `vendor/`, `justfile`.
      (docs/man/ is gitignored — generated on install.)

---

## Phase 2 — Homebrew distribution

- [ ] Create `scripts/fm.rb.tmpl` — Homebrew formula template with `{{VERSION}}` and
      `{{SHA256}}` placeholders. Formula body:
      - `url` pointing to the GitHub release tarball.
      - `depends_on` Go.
      - `install` block: `go build`, `man1.install "docs/man/fm.1"`,
        `generate_completions_from_executable(bin/"fm", "completion")` for
        bash/zsh/fish — Homebrew reruns this on every upgrade.
      - `test` block: `fm --version`.
- [ ] Create `scripts/make-formula.sh`:
      - Usage: `make-formula.sh <tag>` (e.g. `v0.2.0`).
      - Derives tarball URL `https://github.com/backlin/frontmatter/archive/refs/tags/<tag>.tar.gz`.
      - Downloads tarball, computes SHA256, substitutes into template, prints result to
        stdout. Maintainer copies output to `Formula/fm.rb` in the `homebrew-backlin` tap
        repo.
- [ ] Add private just recipe:
      ```
      [private]
      _formula tag:
          scripts/make-formula.sh {{tag}}
      ```
- [ ] Commit: `scripts/fm.rb.tmpl`, `scripts/make-formula.sh`, `justfile`.
