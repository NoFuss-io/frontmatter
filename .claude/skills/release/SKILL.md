---
name: release
description: Release a new version of `fm`
---

## Steps

1. Confirm `main` is clean and CI passes.
2. Run `just release-check` to validate goreleaser config.
3. Draft release notes from git history (see below) and present for user to edit.
4. Decide semver bump based on changes: major (breaking API), minor (new features), patch (fixes only).
5. Tag: `git tag -a vX.Y.Z -m "<release notes>"`
6. Push: `git push origin vX.Y.Z`
7. Confirm GitHub Actions release workflow completes.

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

The tag command reads the file:

```sh
git tag -a vX.Y.Z -m "$(cat release-notes.md)"
```

Delete `release-notes.md` after tagging.

> Goreleaser auto-generates the GitHub release body from the same commit range
> using `.goreleaser.yaml` `changelog` config — no manual step needed for the
> release artifact itself. The curated draft is for the annotated tag message
> and for the user's review.
