baseline = b3be4dd

# Implementation plan — release pipeline fix

Phased rollout of [TARGET_260528_release_fix.md](TARGET_260528_release_fix.md).

Order: external prerequisites first (tap repo, secret), then config migration
on a feature branch, then re-cut the release tag. Earlier phases never depend
on later ones.

---

## Phase A — Tap repo + PAT (manual, outside this repo)

External GitHub state. Not a commit in this repo. Run the commands in any
working directory; they target the org via `gh`.

### A.1 Create the tap repo — [x] done

Requires `gh` authenticated as a user with `Create repositories` permission on
the `NoFuss-io` org.

```sh
# create empty public repo with main as default branch
gh repo create NoFuss-io/homebrew-tap \
  --public \
  --description "Homebrew tap for NoFuss-io tools" \
  --add-readme=false

# clone it somewhere outside the fm checkout
cd /tmp
gh repo clone NoFuss-io/homebrew-tap
cd homebrew-tap

# placeholder so Formula/ exists before goreleaser writes into it
mkdir Formula
touch Formula/.gitkeep
git add Formula/.gitkeep
git commit -m "chore: initial Formula directory placeholder"
git push -u origin main
```

Verify:

```sh
gh api repos/NoFuss-io/homebrew-tap/contents/Formula | jq '.[].name'
# expect: ".gitkeep"
```

### A.2 Mint the fine-grained PAT — [x] done

Manual, browser only. CLI cannot create fine-grained PATs.

1. Visit https://github.com/settings/personal-access-tokens/new.
2. Token name: `homebrew-tap-publish` (or similar).
3. Resource owner: `NoFuss-io` (requires org owner approval if the org
   restricts fine-grained PATs — may queue for an admin to approve).
4. Repository access: **Only select repositories** → `NoFuss-io/homebrew-tap`.
5. Repository permissions:
   - `Contents`: **Read and write**.
   - `Metadata`: Read (auto).
6. Expiration: longest allowed by org policy.
7. Generate, copy the `github_pat_...` string — it is shown once.

### A.3 Install PAT as repo secret — [x] done

```sh
# from anywhere; gh figures out the repo from the --repo flag
gh secret set HOMEBREW_TAP_GITHUB_TOKEN \
  --repo NoFuss-io/frontmatter \
  --body 'github_pat_PASTE_HERE'

# verify
gh secret list --repo NoFuss-io/frontmatter | grep HOMEBREW_TAP_GITHUB_TOKEN
```

Expect a row showing the secret name and a recent updated timestamp.

### A.4 Smoke test the PAT outside the workflow — [ ] todo

Sanity-check the token before relying on it from CI.

```sh
GITHUB_TOKEN=<the PAT> gh api repos/NoFuss-io/homebrew-tap \
  | jq '{name, default_branch, permissions}'
```

Expect `default_branch: "main"` and write permission visible. If this returns
`401`, the PAT is wrong — fix before continuing.

- Commit: none (external state only).

## Phase B — Migrate `.goreleaser.yaml` off `brews`

Touches a single file in this repo. Lands on a feature branch, PRs into
`main`.

- [x] Read the goreleaser deprecation page
      (https://goreleaser.com/deprecations#brews) and pick the current
      supported block name for CLI formula publishing.
      → Successor is `homebrew_casks:` (not `homebrew_formulas:` — that
      was also deprecated in v2.10 alongside `brews:`). Schema is
      declarative (`binaries`, `manpages`, `completions`), not Ruby DSL.
- [x] Replace the `brews:` block in `.goreleaser.yaml` with the new block.
      Note: schema change is **not** a verbatim preservation — `install:`
      Ruby block → `binaries: [fm]` + `manpages: [docs/man/fm.1]` +
      `completions: {bash, zsh, fish}`. `test:` field has no cask analog
      and was dropped. `directory:` changed from `Formula` → `Casks` per
      cask convention. `license:` field not supported by cask block
      (cask file format embeds it elsewhere) — dropped.
- [x] Validate locally — `just release-check` clean after fixing
      `binary` → `binaries` (singular was deprecated in v2.12.6).
- [x] Dry-run snapshot — file now lands at
      `dist/homebrew/Casks/fm.rb` (not `Formula/`). Inspected output:
      proper `cask "fm" do` block with darwin+linux/intel+arm sha256s,
      `binary "fm"`, `manpage`, and bash/zsh/fish completions wired.
- [ ] Open PR, merge once CI green.
- [ ] Commit: `release: migrate goreleaser brews block off deprecated name`.

> **Tap repo consequence**: Phase A created `Formula/.gitkeep` but the
> generated cask will be pushed to `Casks/fm.rb`. The `Formula/` dir in
> the tap becomes vestigial. Either leave it (harmless) or delete in a
> follow-up commit on the tap repo.

## Phase C — Re-cut release as `v0.2.2`

Operational. No code changes beyond Phase B already on `main`.

- [ ] Confirm `main` clean and up to date:
      `git checkout main && git pull --ff-only origin main && git status`.
- [ ] Confirm secret present:
      `gh secret list --repo NoFuss-io/frontmatter | grep HOMEBREW_TAP_GITHUB_TOKEN`.
- [ ] Final sanity sweep: `just lint && just test`.
- [ ] Tag: `git tag -a v0.2.2 -m "Re-cut release with homebrew tap publish"`.
- [ ] Push tag: `git push origin v0.2.2`.
- [ ] Watch `release.yml` from the Actions tab. Expect:
      - All build targets green.
      - `homebrew formula` step shows `writing formula=dist/homebrew/Formula/fm.rb`
        followed by a `committed` line referencing `NoFuss-io/homebrew-tap`,
        not the previous `401 Bad credentials` error.
- [ ] Verify the GitHub Release page populated:
      https://github.com/NoFuss-io/frontmatter/releases/tag/v0.2.2.
- [ ] Verify the formula commit in the tap repo:
      `gh api repos/NoFuss-io/homebrew-tap/contents/Formula/fm.rb | jq .sha`.
- [ ] End-to-end install check (on a machine with Homebrew):
      `brew untap nofuss-io/tap 2>/dev/null; brew install nofuss-io/tap/fm && fm --version`
      should print `v0.2.2`.
- [ ] Commit: none — this phase is operational, not code.

## Phase D — Tidy old `v0.2.1` release page (optional)

The `v0.2.1` GitHub Release exists but has no matching formula. Either:

- [ ] Leave it as-is — users find `v0.2.2` first on the Releases page, the
      `v0.2.1` tag still has working archives for anyone who pinned it.
- [ ] Or annotate the `v0.2.1` Release body with a note:
      "Homebrew publish failed for this tag — use v0.2.2 or later for
      `brew install`."
      Done via `gh release edit v0.2.1 --notes-file ...` or the UI.

Recommend the second — costs nothing and prevents confusion.

- Commit: none.

## Out of scope

- Cosign signing, SLSA attestation, additional package managers, Docker
  images. See target §"Out of scope".
