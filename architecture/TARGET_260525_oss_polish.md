baseline = 5e8787b

# Target state — open-source polish

Five scoped tracks to lift `fm` toward gold-standard OSS hygiene. Each track is
independent and can land in its own PR. Governance is sized for a
single-maintainer hobby project routed through GitHub — no community
bureaucracy, just the minimum that lets a drive-by contributor know what to
expect. Documentation hosting will be handled manually outside this target.

Supersedes [TARGET_260511_homebrew.md](TARGET_260511_homebrew.md): the Homebrew
formula now flows through goreleaser as part of §2 below; the manual
`scripts/make-formula.sh` / `scripts/fm.rb.tmpl` approach is dropped.

---

## 1. CI/release

GitHub Actions covering lint, test, and tagged releases. No releases exist yet
(`git tag -l` is empty); first tag will be `v0.1.0` once these workflows land
green.

### `.github/workflows/ci.yml` (new)

- Triggers: `push` on any branch, `pull_request` targeting `main`.
- Single job on `ubuntu-latest`, Go 1.23.x via `actions/setup-go@v5` with
  module + build cache enabled.
- Steps: checkout, setup-go, install `just` via
  `extractions/setup-just@v3`, install `golangci-lint` via
  `golangci/golangci-lint-action@v6` (see §3), `just lint`, `just test`.
- Concurrency group keyed by branch so older runs cancel on push.

### `.github/workflows/release.yml` (new)

- Trigger: `push` of tags matching `v*.*.*`.
- Permissions: `contents: write` for release upload. No id-token/cosign for now.
- Steps: checkout with `fetch-depth: 0` (goreleaser needs full history),
  setup-go, `goreleaser/goreleaser-action@v6` with `args: release --clean`.
- Secrets consumed:
  - `GITHUB_TOKEN` (default, for the GitHub Release).
  - `HOMEBREW_TAP_GITHUB_TOKEN` — fine-grained PAT with `contents:write` on the
    `NoFuss-io/homebrew-tap` repo, used by the `brews` block in §2.

### `.goreleaser.yaml` (new)

- `project_name: fm`.
- `builds`: single `main: ./cmd/fm`, binary `fm`, targets `linux/amd64`,
  `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`. `ldflags`
  injects `-s -w -X main.Commit={{.ShortCommit}}` and (see below) a version
  string.
- `archives`: `tar.gz` for unix, `zip` for windows. `files`: `LICENSE`,
  `README.md`, `docs/manual.md`, `docs/man/fm.1` (generated — see §3 tooling
  note), `completions/*` (generated).
- `checksum`: SHA-256 manifest, default name template.
- `changelog`: `use: github`, group commits by Conventional Commit prefix
  (`feat`, `fix`, `perf`, `refactor`), exclude `chore`, `docs`, `test`, merge
  commits.
- `release`: `draft: false`, `prerelease: auto` (based on tag suffix).
- `brews`: configured per §2.

### Version wiring

`cmd/fm/main.go` exposes two package-level `string` vars, both injectable via
`-ldflags -X`:

```go
var (
    Version = "dev"
    Commit  = ""
)
```

The `Semver` struct is dropped — the canonical version string comes from the
git tag (`v0.2.1`), not a hand-edited literal. Goreleaser injects via
`-X main.Version={{.Version}} -X main.Commit={{.ShortCommit}}`.

For `go install github.com/nofuss-io/frontmatter/cmd/fm@v0.2.1` builds (no
ldflags), `init()` falls back to `runtime/debug.ReadBuildInfo()`:

- `Version` ← `bi.Main.Version` if non-empty and not `(devel)`.
- `Commit` ← first 7 chars of the `vcs.revision` setting.

Local `go build` with no tag and dirty tree shows `dev` / empty — fine for
hacking. `goreleaser` overrides both. `go install @vX.Y.Z` gets `vX.Y.Z` and
the SHA free.

`justfile` `build` and `install` recipes pass
`-X main.Version=$(git describe --tags --always --dirty) -X main.Commit=$(git rev-parse --short HEAD)`
so locally-installed builds also get meaningful values.

### `justfile` additions

- `release-snapshot`: `goreleaser release --snapshot --clean` (local dry-run,
  artifacts go to `dist/`).
- `release-check`: `goreleaser check` (validates `.goreleaser.yaml`).

---

## 2. Distribution

Three install paths: Homebrew tap, `go install`, prebuilt binaries from GitHub
Releases. All three follow from the goreleaser config in §1.

### 2a. Homebrew tap

Replaces the manual `make-formula.sh` flow from `TARGET_260511_homebrew.md`.
The formula is published automatically on each tagged release.

**`.goreleaser.yaml`** — `brews` entry:

- `repository`: `owner: NoFuss-io`, `name: homebrew-tap`.
- `directory: Formula`.
- `homepage`, `description`, `license: MIT` filled from this repo.
- `install`: `bin.install "fm"`, `man1.install "docs/man/fm.1"`, generate +
  install completions via
  `generate_completions_from_executable(bin/"fm", "completion")` once the
  completion subcommand exists (see §3 — completions). If completions are not
  yet implemented when the first release ships, omit that line and add it in
  the release that introduces the subcommand.
- `test`: `assert_match "fm", shell_output("#{bin}/fm --version")`.

**Tap repo (one-time, manual setup):**

- Create `github.com/NoFuss-io/homebrew-tap` (public, empty except for a
  `Formula/` directory and a short README).
- Mint a fine-grained PAT scoped to that single repo with `contents:write`,
  store as `HOMEBREW_TAP_GITHUB_TOKEN` in `fm` repo secrets.

**End-user install:**

```sh
brew install nofuss-io/tap/fm
```

### 2b. `go install`

Already works — `cmd/fm` is a `main` package. Just needs documenting (see §4).
The injected version will be empty for `go install` builds; that's acceptable.

```sh
go install github.com/nofuss-io/frontmatter/cmd/fm@latest
```

### 2c. Prebuilt binaries

Implicit via goreleaser archives uploaded to the GitHub Release page. README
points users there (see §4).

---

## 3. Tooling

Lock down toolchain, add fuzz coverage on the parser, and tighten editor/CI
hygiene.

### `golangci-lint`

Replaces the bare `staticcheck` call in `justfile`.

**`.golangci.yml`** (new):

- `version: "2"`.
- Linters enabled: `staticcheck`, `govet`, `errcheck`, `unused`, `ineffassign`,
  `gosimple`, `unconvert`, `misspell`, `gofmt`, `goimports`.
- `gofmt`/`goimports`: enforce import grouping (`std`, third-party, local
  prefix `github.com/nofuss-io/frontmatter`).
- `staticcheck.checks`: `["all", "-ST1000"]` (skip "missing package comment").

**`justfile`** `lint` target rewritten:

```
lint:
    go fmt ./...
    golangci-lint run ./...
```

`go vet` is covered by golangci-lint's `govet` linter; the standalone call is
removed.

### Fuzz tests

`internal/parse.go` parses untrusted DSL — natural fuzz target.

**`internal/parse_fuzz_test.go`** (new):

- `FuzzParseProgram`: seed corpus from `internal/parse_test.go` inputs plus
  `docs/tutorial_script.sql` if present.
- `FuzzParseExpr`: seed corpus from expression cases in `parse_test.go`.
- Both fuzz targets assert: `ParseProgram`/`ParseExpr` returns either a valid
  AST or an error; never panics. No structural assertions on accepted inputs.

CI runs a short fuzz smoke in `ci.yml`:
`go test -run=^$ -fuzz=. -fuzztime=30s ./internal/...`
gated to `push` events on `main` only, so PRs stay fast.

### Shell completions and man page generation

Goreleaser archive expects `docs/man/fm.1` and `completions/*`. Neither is
auto-generated today.

- **Man page**: add a `just man` recipe that produces `docs/man/fm.1` from
  the existing manual (either via `pandoc docs/manual.md -s -t man` or a
  hand-maintained roff source). `docs/man/` is already git-ignored — keep it
  that way and have goreleaser regenerate it during release. Add the same
  generation step to `release.yml` before the goreleaser action.
- **Completions**: add a `fm completion {bash|zsh|fish}` subcommand to
  `cmd/fm/main.go`. Generate `completions/fm.bash`, `completions/fm.zsh`,
  `completions/fm.fish` during release via a `before.hooks` block in
  `.goreleaser.yaml`. If completions slip past v0.1.0, drop the corresponding
  goreleaser lines and reintroduce them in the release that adds the
  subcommand.

### Editor + dependency hygiene

**`.editorconfig`** (new):

- `*.go`: tab indent, LF, trim trailing whitespace, final newline.
- `*.{yml,yaml,md,sh,toml,json}`: 2-space indent, LF, trim, final newline.

**`.github/dependabot.yml`** (new):

- Ecosystem `gomod`, schedule weekly, target branch `main`, max 5 open PRs,
  commit-message prefix `chore(deps)`.
- Ecosystem `github-actions`, schedule weekly, max 5 open PRs.

### Pre-commit

The repo CLAUDE.md mentions a pre-commit hook, but no `.pre-commit-config.yaml`
is checked in. Check whether the hook lives in `.git/hooks/` and is
machine-local; if so, add a `.pre-commit-config.yaml` running `just lint` and
`just test` so contributors get the same checks. (No-op if no hook exists.)

---

## 4. README polish

Lift the README from "works" to "scannable in 30 seconds".

### Badges

Add a row of badges immediately under the H1, before the tagline:

- CI status:
  `https://github.com/NoFuss-io/frontmatter/actions/workflows/ci.yml/badge.svg`
- Latest release:
  `https://img.shields.io/github/v/release/NoFuss-io/frontmatter`
- License:
  `https://img.shields.io/github/license/NoFuss-io/frontmatter`
- Go Reference:
  `https://pkg.go.dev/badge/github.com/nofuss-io/frontmatter`
- Go Report Card:
  `https://goreportcard.com/badge/github.com/nofuss-io/frontmatter`

### Status line

One line under the badges: `**Status:** Alpha — single-maintainer project,
breaking changes possible until v1.0.`

### Installation section

Rewrite to lead with the easy paths, demote source build to a footnote:

1. **Homebrew**: `brew install nofuss-io/tap/fm`
2. **`go install`**: `go install github.com/nofuss-io/frontmatter/cmd/fm@latest`
3. **Prebuilt binaries**: link to the GitHub Releases page.
4. **From source** (current `git clone … just install` block, kept but moved
   to the bottom). Swap the clone URL from `git@github.com:…` to
   `https://github.com/NoFuss-io/frontmatter.git` so users without SSH
   configured can copy-paste.

Move the `⚠️ Note: I recommend only using fm on Vaults under version control`
warning to sit immediately above the install commands so it's read before any
command is run.

### Comparison

New short section after "In a nutshell": one paragraph plus a small table
contrasting `fm` with `yq`, `dasel`, and plain `awk`/`sed`. Highlight: SQL
DSL, multi-file globs, type system, Obsidian wikilink support.

### Roadmap

New short section linking to `architecture/FEATURE_WINDOWS.md` and any other
target docs. Bulleted, no dates.

### Contributing

One paragraph deferring to "open an issue first" — a real `CONTRIBUTING.md`
is deferred until a community exists.

---

## 5. Governance

Single-maintainer, hobby-pace, everything through GitHub. The goal is to set
expectations so a drive-by contributor doesn't waste effort, not to install a
foundation.

### `CONTRIBUTING.md` (new, short)

One screen, no checklists. Cover:

- "This is a hobby project; I respond when I can, sometimes slowly."
- **Before opening a PR**: open an issue first to check the change is wanted.
  Unsolicited large PRs may be closed unmerged.
- **Local checks**: `just lint && just test` must pass. CI will re-run them.
- **Commit style**: Conventional Commits (`feat:`, `fix:`, `refactor:`,
  `docs:`, `chore:`, `test:`). Goreleaser groups the changelog from these
  prefixes (§1).
- **Scope**: keep PRs focused — one logical change per PR. Link the issue.
- Link to `architecture/BASELINE.md` and `AGENTS.md` for orientation.

### `SECURITY.md` (new, short)

- Supported versions: latest tagged release only.
- **Reporting**: use GitHub's private vulnerability reporting
  (`Security` tab → `Report a vulnerability`). Enable this in repo settings
  (Settings → Code security → Private vulnerability reporting).
- Response SLA: best-effort, no guarantee. Acknowledge within 14 days.
- Threat model note: `fm` reads/writes local Markdown files and runs a
  hand-rolled DSL parser. The interesting class of bug is a malicious
  frontmatter file or DSL input crashing/hanging the parser — fuzzing in §3
  is the primary mitigation.

### `.github/ISSUE_TEMPLATE/` (new)

Two templates only — keep the chooser short.

- **`bug_report.yml`** — fields: `fm --version` output, OS, minimal
  reproducer (input file + command + observed vs expected).
- **`feature_request.yml`** — fields: use case, proposed syntax (if any),
  alternatives considered.

Plus **`.github/ISSUE_TEMPLATE/config.yml`**:
- `blank_issues_enabled: false`
- `contact_links`: one entry pointing questions/discussion at GitHub
  Discussions (enable Discussions in repo settings as part of this track).

### `.github/PULL_REQUEST_TEMPLATE.md` (new)

Three sections, ≤10 lines total:

- **Summary** (1–2 lines).
- **Linked issue** (`Closes #N`).
- **Checklist**: tests added/updated, `just lint && just test` passes, docs
  updated if behavior changed.

### `CODEOWNERS` (new, single line)

```
* @backlin
```

Auto-assigns the maintainer to every PR. Cheap and useful even as the sole
contributor because GitHub shows a clear "review requested" state.

### `CODE_OF_CONDUCT.md`

Skipped intentionally. With no community to govern, the file is theater. Add
it when there are recurring contributors or a Discussions thread that needs
moderation rules, not before.

### Repo-settings checklist (one-time, manual)

Not a file, but list it here so it's not forgotten:

- Enable **Private vulnerability reporting** (Settings → Code security).
- Enable **Dependabot alerts** and **Dependabot security updates** (Settings
  → Code security). Dependabot version updates are configured in §3.
- Enable **GitHub Discussions** (Settings → Features) — referenced by the
  issue-template `config.yml`.
- Branch protection on `main`: require status checks (`ci.yml`), require
  linear history, disallow force pushes. Allow the maintainer to bypass —
  this is a solo project, not a team.

---

## Out of scope

- Hosted documentation site (GitHub Pages, mkdocs, etc.) — manual push.
- Submission to `homebrew-core` — premature; the tap is enough.
- macOS code signing / notarization — deferred until users ask.
- Other package managers (apt, AUR, nixpkgs, scoop, chocolatey) — deferred.
- Docker image — non-goal for a single-file Go binary.
- Coverage upload to codecov/coveralls — deferred until the test suite
  stabilizes.
- Sigstore/cosign signing of release artifacts — deferred.
- Any new product features, query forms, or behavioral changes.
