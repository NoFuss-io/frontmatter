---
name: release
description: Release a new version of `fm`
---

## Steps

1. Confirm `main` is clean and CI passes.
2. Run `just release-check` to validate goreleaser config.
3. Draft release notes from git history (see below) and present for user to edit.
4. Decide semver bump based on changes: major (breaking API), minor (new features), patch (fixes only).
5. Bump the plugin version. The plugin manifests (`plugin.json`,
   `.claude-plugin/plugin.json`, `gemini-extension.json`) must carry the same
   version as the git tag, because agent hosts use it to detect plugin updates.
   `main` is protected (PR + `lint-and-test` required), so:
   ```sh
   git switch -c release-vX.Y.Z origin/main
   just set-version X.Y.Z        # no `v`
   git commit -am "release: vX.Y.Z"
   git push -u origin release-vX.Y.Z
   gh pr create --fill
   ```
   Wait for the PR to be merged, then `git switch main && git pull`.
6. Verify, then tag the merged commit on `main`:
   ```sh
   just check-version X.Y.Z
   git tag -a vX.Y.Z -m "$(cat release-notes.md)"
   ```
7. Push the tag: `git push origin vX.Y.Z`
8. Confirm the GitHub Actions release workflow completes. Before building, it runs
   `just check-version` against the tag and fails the release if the
   manifests do not match.

## Release notes draft

Fetch commits since last tag:

```sh
git log $(git describe --tags --abbrev=0)..HEAD \
  --pretty=format:"%s" \
  --no-merges | grep -Ev "^(chore|docs|test|refactor|ci|release)(\(.+\))?:"
```

Group under **Features** and **Bug fixes**. Rewrite commit subjects into
user-facing language: drop conventional-commit prefix, expand abbreviations,
describe the user-visible effect rather than the implementation detail.

Write the draft to `release-notes.md` in the repo root. Include the target
version on the first line (e.g. `v0.3.0`), then tell the user to edit it
before proceeding.

Before tagging, verify the version in `release-notes.md` matches the intended
tag — guard against a stale file left over from a previous aborted release:

```sh
head -1 release-notes.md   # must match the tag you are about to create
```

The tag command in step 6 reads the file.

Delete `release-notes.md` after tagging.

> Goreleaser auto-generates the GitHub release body from the same commit range
> using `.goreleaser.yaml` `changelog` config — no manual step needed for the
> release artifact itself. The curated draft is for the annotated tag message
> and for the user's review.
