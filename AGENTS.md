# Repository guide

`fm` is a CLI tool for querying and mutating YAML frontmatter in Markdown files,
targeting knowledge management workflows (e.g. Obsidian vaults).

For a full architectural overview see [architecture/BASELINE.md](architecture/BASELINE.md).

## Directory structure

```
fm/
├── cli/                  # All application source (Go, package main)
│   ├── main.go           # Command definitions: select, update, alter
│   ├── core.go           # Data model (File), file I/O, matching, mutation
│   └── parse.go          # Parsers for fields, comparisons, assignments, expressions
├── docs/
│   ├── syntax.md         # Complete CLI syntax reference
│   ├── tutorial.md       # Step-by-step tutorial walkthrough
│   └── tutorial/         # Sample Markdown recipe files used by the tutorial
├── architecture/
│   └── BASELINE.md       # High-level architecture baseline (services, types, workflows)
├── vendor/               # Vendored Go dependencies — do not edit
├── go.mod / go.sum       # Go module definition and checksums
├── justfile              # Build automation: build, install, lint, vendor, dev
└── README.md             # Project overview and quick-start examples
```

## Key documentation

| File | Purpose |
|:-----|:--------|
| [README.md](README.md) | Quick start, goals, example commands |
| [docs/syntax.md](docs/syntax.md) | Full reference: all commands, field types, operators, expressions |
| [docs/tutorial.md](docs/tutorial.md) | Guided walkthrough using the recipe collection in `docs/tutorial/` |
| [architecture/BASELINE.md](architecture/BASELINE.md) | Architecture: data model, type system, parsing layer, workflows |
