# Contributing

Thanks for helping improve `fm`! Bug reports, feature ideas and pull requests
are all welcome, and I aim to respond quickly.

## Before opening a PR

**Open an issue first** for anything bigger than a small fix. A short
description of the change and the use case is enough, and it lets us agree
on shape and scope before you put in the work.

## Local checks

```sh
just lint
just test
```

Both must pass. CI re-runs the same checks on every push.

They can also run automatically with a pre-commit git hook, installed by:

```sh
just setup
```

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
