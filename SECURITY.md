# Security Policy

## Supported versions

Only the latest tagged release is supported. There is no LTS line.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting**:

1. Go to the [Security tab](https://github.com/NoFuss-io/frontmatter/security).
2. Click **Report a vulnerability**.
3. Fill in the form. Reports stay private until I publish an advisory.

Please do not file public issues for security problems.

## Response

This is a single-maintainer hobby project, so response is best-effort. I aim
to acknowledge reports within 14 days. Fix turnaround depends on severity and
my availability.

## Threat model

`fm` reads and writes local Markdown files and parses an SQL-style DSL
provided by the invoking user. The interesting class of bug is a malicious
frontmatter file or DSL input that crashes, hangs, or causes incorrect
mutation of unrelated files.

Mitigations in place:

- Hand-rolled parser fuzzed in CI (`internal/parse_fuzz_test.go`).
- Frontmatter decoded by `gopkg.in/yaml.v3`; we do not execute arbitrary
  YAML tags.
- No network I/O, no shell execution, no plugin loading.

Out of scope:

- Maliciously crafted Markdown that exploits a downstream reader (e.g.
  Obsidian). `fm` aims to round-trip whatever YAML the user has, faithfully.
- Filesystem-level attacks (symlink races on shared trees, etc.).
