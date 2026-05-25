# `fm`: Markdown frontmatter batch editor

[![CI](https://github.com/NoFuss-io/frontmatter/actions/workflows/ci.yml/badge.svg)](https://github.com/NoFuss-io/frontmatter/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/NoFuss-io/frontmatter)](https://github.com/NoFuss-io/frontmatter/releases)
[![License](https://img.shields.io/github/license/NoFuss-io/frontmatter)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/nofuss-io/frontmatter.svg)](https://pkg.go.dev/github.com/nofuss-io/frontmatter)
[![Go Report Card](https://goreportcard.com/badge/github.com/nofuss-io/frontmatter)](https://goreportcard.com/report/github.com/nofuss-io/frontmatter)

**Status:** Alpha — single-maintainer project, breaking changes possible until v1.0.

Harmonize, refactor, and batch edit YAML frontmatter using SQL expressions.

Designed for Markdown in general and [Obsidian](https://obsidian.md) in particular.
Vaults are modelled as big jagged tables where documents are rows and [fields](https://obsidian.md/help/properties) are columns.

- [Manual](./docs/manual.md)
- [Tutorial](./docs/tutorial/tutorial.md)
- [Blog post](https://nofuss.io/en/frontmatter/)

## Installation

⚠️ 👉 **Note:** I recommend only using `fm` on Vaults under version control or backup,
in case you mess up. Like most terminal commands, _it's got no undo feature._

### Homebrew (macOS / Linux)

```sh
brew install nofuss-io/tap/fm
```

### `go install`

```sh
go install github.com/nofuss-io/frontmatter/cmd/fm@latest
```

### Prebuilt binaries

Download the archive for your platform from the
[Releases page](https://github.com/NoFuss-io/frontmatter/releases), extract,
and place the `fm` binary on your `PATH`.

### From source

Requirements: Git, [Go](https://go.dev/), and [Just](https://just.systems/):

```sh
git clone https://github.com/NoFuss-io/frontmatter.git
cd frontmatter
just install
just install-skill
```


## In a nutshell

```sh
fm \
    SELECT [expression], ... \
    FROM [files] \
    WHERE [condition] \
    SORT BY [expression], ... \
    LIMIT [number]
```

```sh
fm UPDATE [files] SET [assignment], ... WHERE [condition];
fm ALTER [files] DROP [field], ... WHERE [condition];
fm ALTER [files] RENAME [field] TO [field], ... WHERE [condition];
```

For example:

```sh
fm select vegan, vegetarian from recipes/*
```

```
filename                                          vegan  vegetarian
--------                                          -----  ----------
Amaretto-grädde.md                                false  true
Apelsinkaka.md
Apfelstrudel.md
Baba ganoush.md                                   true
Bacon Jam.md
Baconlindad fläskfilé med örter och senapssås.md
Banan- och havrecookies.md                        true
Basilikakyckling.md
Bechamelsås.md                                    false  true
Belugabolognese.md                                true
```

## Syntax

SQL syntax follows BigQuery dialect but implements only a subset of it (see [manual](./docs/manual.md)).

Notably missing:

- Functions (in the roadmap)
- Joins
- Aggregations
- Window functions


### Notable additions

#### Shorthand type checking and casting

Only return documents with a property of a given type:

```sh
fm 'select * from recipes/* where `Prep time`:int'
```
```sh
fm 'select * from recipes/* where `Prep time` and not `Prep time`:int'
```

Cast field to type:

```sh
fm 'update recipes/* set `Prep time`:int where `Prep time`'
```

[Read more](./docs/manual.md#Fields).

## Compared to other tools

| Tool             | Scope                          | Multi-file globs | Type-aware mutations | Obsidian wikilinks |
|:-----------------|:-------------------------------|:-----------------|:---------------------|:-------------------|
| `fm`             | Markdown frontmatter, SQL DSL  | ✅               | ✅                   | ✅                 |
| `yq`             | Any YAML, jq-style expressions | manual loop      | partial              | ❌                 |
| `dasel`          | Multi-format (yaml/json/toml)  | manual loop      | partial              | ❌                 |
| `awk` / `sed`    | Line-oriented text             | ✅               | ❌                   | ❌                 |

`fm` trades generality for ergonomics in one workflow: querying and mutating
YAML frontmatter across many Markdown files at once, with a syntax that reads
like SQL and round-trips Obsidian-specific link types.

## Roadmap

- Window and aggregate clauses — see [architecture/FEATURE_WINDOWS.md](./architecture/FEATURE_WINDOWS.md).
- Functions in expressions.
- See open [Issues](https://github.com/NoFuss-io/frontmatter/issues) and
  [Discussions](https://github.com/NoFuss-io/frontmatter/discussions) for
  what's being kicked around.

## Contributing

This is a hobby project — open an issue before sending a PR. See
[CONTRIBUTING.md](./CONTRIBUTING.md) for details.
