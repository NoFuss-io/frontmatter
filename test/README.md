# Tests
Currently only integration tests.

Plausible future content:

| Folder              | Purpose                                                                |
|---------------------|------------------------------------------------------------------------|
| test/e2e/           | Full workflows across multiple invocations (write → query → mutate)    |
| test/fixtures/      | Shared markdown corpora reused by several test packages                |
| test/golden/        | Golden snapshots if you split them out of integration cases            |
| test/fuzz/          | Fuzz seed corpus + harnesses (parse.go is prime fuzz target)           |
| test/bench/         | Benchmark workloads (large vaults, deep glob trees)                    |
| test/regression/    | Pinned bug repros — one dir per past issue, never edited after merge   |
| test/smoke/         | Tiny "does the binary start and exit 0" checks, run post-install       |
| test/conformance/   | Compare fm output against reference tool (yq, custom parser) on shared inputs |
| test/vaults/        | Real-world sample Obsidian vaults (sanitized), exercise messy YAML     |
| test/perf/          | Performance baselines with thresholds (regression-detect on CI)        |
| test/snapshots/     | CLI help text / error message snapshots, separate from data tests      |
| test/compat/        | Old fm-version output goldens, ensure backward-compat                  |
