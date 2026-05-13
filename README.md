# Markdown frontmatter editor `fm`

CLI that harmonizes, refactors, and batch edits yaml frontmatter in markdown files
using SQL expressions.

Great utility tool for [Obsidian](https://obsidian.md).


## Agentic installation

```sh
git clone git@github.com/backlin/fm.git
cd fm
just install-skill
```

## tl;dr

- [Complete syntax](./docs/syntax.md)
- [Tutorial](./docs/tutorial.md)

In a nutshell:

```sh
fm \
    SELECT <fields> \
    FROM <files> \
    WHERE <conditions> \
    SORT BY <fields> \
    LIMIT <number>
```

```sh
fm UPDATE <files> SET <assignments> WHERE <conditions>
```

```sh
fm ALTER <files> DROP <fields> WHERE <conditions>
```

### Example

`fm select vegan, vegetarian from *`

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


## Installation

```sh
go install github.com/backlin/frontmatter
```


# For next agentic session

Verbose output mode.
Assignment from other field.
Unit tests.