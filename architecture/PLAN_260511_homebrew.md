# Plan: Homebrew distribution

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
      - Derives tarball URL `https://github.com/nofuss-io/fm/archive/refs/tags/<tag>.tar.gz`.
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
