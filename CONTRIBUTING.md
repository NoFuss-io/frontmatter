# Contributing

`fm` is a single-maintainer hobby project.
I try to be helpful but respond when I can.

## Before opening a PR

**Open an issue first.** A short description of the change and the use case
is enough. Unsolicited large PRs may be closed unmerged because shape and
scope haven't been agreed.

## Local checks

```sh
just lint
just test
```

Both must pass. CI re-runs the same checks on every push.

## Commit style

[Conventional Commits](https://www.conventionalcommits.org). The release
pipeline groups the changelog by prefix:

- `feat:` user-visible new behaviour
- `fix:` user-visible bug fix
- `perf:` measurable performance change
- `refactor:` internal cleanup, no behaviour change
- `docs:` documentation only
- `test:` test-only changes
- `chore:` everything else (deps, tooling, CI)

## PR scope

One logical change per PR. Link the issue (`Closes #N`). Mixed PRs are slower
to review and harder to revert.

## Orientation

- [README.md](README.md) — what `fm` is and how to use it.
- [docs/manual.md](docs/manual.md) — full DSL reference.
- [architecture/BASELINE.md](architecture/BASELINE.md) — package layout, data
  model, parser/evaluator overview.
- [AGENTS.md](AGENTS.md) — repository guide aimed at AI coding assistants;
  also a fast read for humans.

## Questions, not bugs

Use [GitHub Discussions](https://github.com/NoFuss-io/frontmatter/discussions)
for usage questions, design discussion, or feature ideation. Issues are for
concrete bugs and accepted feature requests.
