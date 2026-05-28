baseline = d1cb244

# Implementation plan — open-source polish

Phased rollout of [TARGET_260525_oss_polish.md](TARGET_260525_oss_polish.md).
Each phase is its own commit so the bisect log stays useful and any phase can
be reverted in isolation.

Order is bottom-up: local tooling → governance → CI → release → distribution
→ README → tag. Earlier phases never depend on later ones. The first real
release is `v0.2.1` at the end of Phase H.

---

## Phase A — Local tooling

Foundation. Touches no shared state, lands first so subsequent CI work picks
up the new lint config.

- [x] Install `golangci-lint` locally (Go 1.23 compatible version, currently
      v1.62.x). Document the version in `.golangci.yml` `version: "2"`.
- [x] Add `.golangci.yml` with linters: `staticcheck`, `govet`, `errcheck`,
      `unused`, `ineffassign`, `gosimple`, `unconvert`, `misspell`, `gofmt`,
      `goimports`. `goimports` configured with local prefix
      `github.com/nofuss-io/frontmatter`.
- [x] Rewrite `justfile` `lint` target:
      `go fmt ./...` + `golangci-lint run ./...`. Drop the standalone
      `go vet` and `staticcheck` calls (golangci-lint covers both).
- [x] `.editorconfig` covering `*.go` (tab) and `*.{yml,yaml,md,sh,toml,json}`
      (2-space).
- [x] Add `internal/parse_fuzz_test.go` with `FuzzParseProgram` and
      `FuzzParseExpr`. Seed corpora from existing parse tests + the tutorial
      script if it exists. Assert no panic; accept any error.
- [x] Run `just lint && just test && go test -run=^$ -fuzz=Fuzz -fuzztime=10s ./internal/...`
      locally — all green.
- [x] Add `release-snapshot` and `release-check` recipes to `justfile`
      (no-op until Phase E adds `.goreleaser.yaml`, but the recipe shells
      are cheap to land now). Or defer to Phase E — author's call.
- [x] Commit: `tooling: adopt golangci-lint, add .editorconfig, add parser fuzz tests`.

## Phase B — Build-info wiring in justfile

Version refactor already landed (commit `d1cb244`). This phase only updates
the `justfile` recipes to inject build-info ldflags.

- [x] `justfile` `build` recipe: pass
      `-ldflags "-X main.Version=$(git describe --tags --always --dirty) -X main.Commit=$(git rev-parse --short HEAD)"`
      so a local `just build` reports a meaningful version.
- [x] `justfile` `install` recipe: same ldflags.
- [x] `justfile` `dev` recipe: leave as-is (`go run` doesn't need version
      string).
- [x] Smoke-check: `just build && ./fm --help` shows
      `v0.2.0-N-gXXXXXXX` or similar plus the SHA.
- [x] Commit: `build: inject Version/Commit via ldflags in justfile`.

## Phase C — Governance files

Plain Markdown additions and repo-settings checklist. Independent of CI.

- [x] `.github/ISSUE_TEMPLATE/bug_report.yml`. Fields: `fm --version`
      output, OS, minimal reproducer (input file + command + observed vs
      expected).
- [x] `.github/ISSUE_TEMPLATE/feature_request.yml`. Fields: use case,
      proposed syntax (if any), alternatives considered.
- [x] `.github/ISSUE_TEMPLATE/config.yml`: `blank_issues_enabled: false`,
      contact link pointing at GitHub Discussions.
- [x] `.github/PULL_REQUEST_TEMPLATE.md`: Summary / Linked issue / Checklist
      (≤10 lines total).
- [x] `CODEOWNERS` (top-level): `* @backlin`.
- [x] `CONTRIBUTING.md`: hobby-pace expectations, open-issue-first rule,
      `just lint && just test` required, Conventional Commit prefixes,
      one-change-per-PR. Link to `architecture/BASELINE.md` and `AGENTS.md`.
- [x] `SECURITY.md`: supported versions = latest tagged release; reporting
      via GitHub private vulnerability reporting; 14-day acknowledge SLA;
      threat-model note (parser fuzzing is the main mitigation).
- [x] Manual repo settings (not files — track in commit message body):
      - Enable Private vulnerability reporting (Settings → Code security).
      - Enable Dependabot alerts + security updates.
      - Enable GitHub Discussions.
      - Branch protection on `main`: require `ci.yml` status check, require
        linear history, disallow force pushes, allow maintainer bypass.
        Defer the status-check requirement until Phase D lands the workflow.
- [x] Commit: `governance: add CONTRIBUTING, SECURITY, CODEOWNERS, issue/PR templates`.

## Phase D — CI workflow

Lint + test on push and PR. Uses the `.golangci.yml` from Phase A.

- [x] `.github/workflows/ci.yml`. Single job on `ubuntu-latest`:
      - `actions/checkout@v4`
      - `actions/setup-go@v5` with `go-version: 'stable'` and `cache: true`
      - `extractions/setup-just@v3`
      - `golangci/golangci-lint-action@v6` with `version: v1.62`
      - `just lint`
      - `just test`
- [x] Concurrency block: `group: ci-${{ github.ref }}`, `cancel-in-progress: true`.
- [x] Push, open a PR, confirm green.
- [x] Add `.github/workflows/ci.yml` to the branch-protection required
      checks list (manual, in Settings → Branches).
- [x] `.github/dependabot.yml`: gomod weekly, github-actions weekly. Both
      with `commit-message: prefix: "chore(deps)"`.
- [x] Commit: `ci: add lint+test workflow and dependabot config`.

## Phase E — goreleaser config

Local-only first; release workflow follows in Phase F. Lets `goreleaser release --snapshot --clean` run clean before pushing a tag.

- [x] `.goreleaser.yaml`:
      - `project_name: fm`.
      - `builds`: `main: ./cmd/fm`, binary `fm`, targets `linux/amd64`,
        `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`,
        `ldflags: -s -w -X main.Version={{.Version}} -X main.Commit={{.ShortCommit}}`.
      - `archives`: tar.gz unix / zip windows, include `LICENSE`,
        `README.md`, `docs/manual.md`, `docs/man/fm.1`, `completions/*`.
      - `checksum`: SHA-256 manifest.
      - `changelog.use: github`, group by Conventional Commit prefix,
        exclude `chore`, `docs`, `test`, merge commits.
      - `release.draft: false`, `release.prerelease: auto`.
- [x] Man-page generation: pick one of:
      (a) hand-maintained roff at `docs/man/fm.1` (un-ignored from
          `.gitignore`), or
      (b) `just man` recipe rendering via `pandoc docs/manual.md -s -t man`
          and a `before.hooks` in `.goreleaser.yaml` calling it.
      Recommend (b) — keeps `docs/man/` git-ignored, regenerates on each
      release.
- [x] Completion subcommand: add `fm completion {bash|zsh|fish}` to
      `cmd/fm/main.go`. Generate `completions/fm.{bash,zsh,fish}` via a
      `before.hooks` block in `.goreleaser.yaml`.
- [x] Validate locally: `just release-check` and
      `just release-snapshot` — confirm `dist/` populated with all
      platform archives + checksums.
- [x] Commit: `release: add goreleaser config, man-page generation, completion subcommand`.

## Phase F — Release workflow + Homebrew tap

External-state phase. Requires the tap repo to exist before the workflow
will succeed.

- [x] Create empty `github.com/NoFuss-io/homebrew-tap` (manual, GitHub UI).
      Add a placeholder `Formula/.gitkeep` so the directory exists.
- [x] Mint a fine-grained PAT scoped to `NoFuss-io/homebrew-tap` with
      `contents:write`. Add as secret `HOMEBREW_TAP_GITHUB_TOKEN` on
      `NoFuss-io/frontmatter`.
- [x] `.goreleaser.yaml` — add `brews` section pointing at
      `NoFuss-io/homebrew-tap`, install binary + man page +
      shell completions, `test: assert_match "fm", shell_output("#{bin}/fm --version")`.
- [x] `.github/workflows/release.yml`:
      - Trigger: `push` of tags matching `v*.*.*`.
      - Permissions: `contents: write`.
      - Steps: checkout (`fetch-depth: 0`), setup-go,
        `goreleaser/goreleaser-action@v6` with `args: release --clean`,
        env: `GITHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`.
- [x] Add `ci.yml` and `release.yml` status badges to README (lands in
      Phase G but flag here).
- [x] Commit: `release: add tag-triggered release workflow with homebrew tap publish`.

## Phase G — README polish

Cosmetic but reader-facing. Lands before the tag so visitors at v0.2.1 see
the polished page.

- [x] Add badge row immediately under `# fm: Markdown frontmatter batch editor`:
      - CI:        `https://github.com/NoFuss-io/frontmatter/actions/workflows/ci.yml/badge.svg`
      - Release:   `https://img.shields.io/github/v/release/NoFuss-io/frontmatter`
      - License:   `https://img.shields.io/github/license/NoFuss-io/frontmatter`
      - Go ref:    `https://pkg.go.dev/badge/github.com/nofuss-io/frontmatter`
      - Go report: `https://goreportcard.com/badge/github.com/nofuss-io/frontmatter`
- [x] Status line under the badges:
      `**Status:** Alpha — single-maintainer project, breaking changes possible until v1.0.`
- [x] Rewrite Installation section:
      1. Homebrew (`brew install nofuss-io/tap/fm`)
      2. `go install github.com/nofuss-io/frontmatter/cmd/fm@latest`
      3. Prebuilt binaries (link to Releases)
      4. From source (existing block, https URL not ssh, moved to bottom).
- [x] Move the `⚠️ Note` about Vault backups to sit immediately above the
      install commands.
- [x] Add **Comparison** section (one paragraph + small table) vs `yq`,
      `dasel`, `awk`/`sed`. Highlight SQL DSL, multi-file globs, type
      system, Obsidian wikilink support.
- [x] Add **Roadmap** section linking `architecture/FEATURE_WINDOWS.md` and
      any other future target docs. Bulleted, no dates.
- [x] Add **Contributing** section: one paragraph, "open an issue first,
      see CONTRIBUTING.md".
- [x] Commit: `docs: README badges, status line, install paths, comparison and roadmap`.

## Phase H — Cut v0.2.1

The payoff.

- [x] Final sanity sweep: `just lint && just test`, no uncommitted changes
      on `main`.
- [x] Refresh `architecture/BASELINE.md` if any structural details drifted
      since the last baseline (paths now `cmd/fm` + `internal/`, not the
      stale `cli/` + `lib/` references). Out of scope for this target but
      worth doing in the same release window.
- Tag: `git tag -a v0.2.1 -m "First proper release"`.
- Push: `git push origin main v0.2.1`.
- Watch `release.yml` complete. Verify:
      - GitHub Release page populated with archives + checksums.
      - `homebrew-tap` repo has a new `Formula/fm.rb` commit.
      - `brew install nofuss-io/tap/fm && fm --version` prints `v0.2.1`.
- Commit: none — this phase is operational, not code.

## Out of scope

- Hosted documentation site, code signing, additional package managers,
  Docker image, codecov upload. See target §"Out of scope".
