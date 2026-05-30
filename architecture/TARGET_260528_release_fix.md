baseline = b3be4dd

# Target state — release pipeline fix

The `v0.2.1` tag pushed cleanly: GitHub Release page, archives, checksums all
present. The Homebrew formula publish step failed with `401 Bad credentials`
because the `HOMEBREW_TAP_GITHUB_TOKEN` secret was empty in the workflow env.
Goreleaser also emitted a `brews` deprecation warning that will become a hard
error in a future major version.

This target restores end-to-end release publish, including the formula push to
`NoFuss-io/homebrew-tap`, and migrates off the deprecated `brews` block. After
this lands a re-cut tag (`v0.2.2`) should produce both a GitHub Release and a
formula commit in the tap repo without manual intervention.

---

## 1. Homebrew tap repo

Must exist before the release workflow can succeed. Owned by the GitHub org
`NoFuss-io`, not the user account. One-time creation, not idempotent.

### `NoFuss-io/homebrew-tap` (new repo, separate from this one)

- Public.
- Default branch `main`.
- Contains a single placeholder file `Formula/.gitkeep` so the `Formula/`
  directory exists before goreleaser tries to write into it.
- README optional; goreleaser does not require it.

### PAT for the formula push

Fine-grained personal access token. Scope:

- Resource owner: `NoFuss-io`.
- Repository access: only `NoFuss-io/homebrew-tap`.
- Repository permissions:
  - `Contents`: Read and write.
  - `Metadata`: Read (auto-included).
- Expiration: pick the longest the org policy allows. Calendar a renewal
  reminder.

Stored as repository secret `HOMEBREW_TAP_GITHUB_TOKEN` on
`NoFuss-io/frontmatter`. Not an org-wide secret — narrowest blast radius.

---

## 2. `.goreleaser.yaml` — migrate off `brews`

Goreleaser v2 deprecated `brews:` in favour of `homebrew_casks:` for casks and
kept formula publishing under a renamed block. Current config still uses
`brews:` and produces:

```
DEPRECATED: brews should not be used anymore, check https://goreleaser.com/deprecations#brews for more info
```

Action: replace the `brews:` block with the current supported equivalent for
CLI formulae per the goreleaser deprecation page. Keep all other fields the
same — repository owner/name/token, install block, test block, directory,
homepage, description, license.

The build, archive, checksum, changelog, and release sections are unchanged.

---

## 3. `.github/workflows/release.yml` — no structural change

The workflow itself is correct. The fix is operational:

- Confirm secret `HOMEBREW_TAP_GITHUB_TOKEN` is set on `NoFuss-io/frontmatter`
  and non-empty.
- Confirm the env block in the workflow still references it (it does at
  baseline).
- No new steps, no permission changes.

---

## 4. Re-cut release

Tag `v0.2.1` is already burned. Two paths:

1. Delete the existing `v0.2.1` GitHub Release (keep the tag), re-run the
   workflow against the same tag. Cleanest if the tap repo was the only
   missing piece.
2. Cut a new patch `v0.2.2` with the goreleaser config migration commit. This
   is the recommended path because (a) the `.goreleaser.yaml` change should be
   on `main` under a tag anyway, and (b) it avoids any chance of goreleaser
   refusing to overwrite the existing release.

Choose path 2.

---

## Out of scope

- Cosign / sigstore signing of artefacts.
- SLSA provenance attestation.
- Additional package managers (apt, scoop, nix).
- Docker image publication.
- Changing the supported platform matrix.
- Renaming the tap repo or moving it under a different org.
