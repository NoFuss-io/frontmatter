# `fm`: Markdown frontmatter batch editor

CLI that harmonizes, refactors, and batch edits yaml frontmatter in markdown files
using SQL expressions.

Compatibile with [Obsidian](https://obsidian.md).

- [Manual](./docs/manual.md)
- [Tutorial](./docs/tutorial/tutorial.md)
- [Blog post](https://nofuss.io/en/frontmatter/)


## Installation

```sh
git clone git@github.com/NoFuss-io/frontmatter.git
cd frontmatter
just install
just install-skill
```

## In a nutshell

```sh
fm \
    SELECT <expression>, ... \
    FROM <files> \
    WHERE <condition> \
    SORT BY <expression>, ... \
    LIMIT <number>
```

```sh
fm UPDATE <files> SET <assignment>, ... WHERE <condition>;
fm ALTER <files> DROP <field>, ... WHERE <condition>;
fm ALTER <files> RENAME <field> TO <field>, ... WHERE <condition>;
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
