# `fm`: Markdown frontmatter batch editor

Harmonize, refactor, and batch edit YAML frontmatter using SQL expressions.

Designed for Markdown in general and [Obsidian](https://obsidian.md) in particular.
Vaults are modelled as big jagged tables where documents are rows and [fields](https://obsidian.md/help/properties) are columns.

- [Manual](./docs/manual.md)
- [Tutorial](./docs/tutorial/tutorial.md)
- [Blog post](https://nofuss.io/en/frontmatter/)



## Installation

Requirements: Git, [Go](https://go.dev/), and [Just](https://just.systems/):

```sh
git clone git@github.com/NoFuss-io/frontmatter.git
cd frontmatter
just install
just install-skill
```

⚠️ 👉 **Note:** I recommend only using `fm` on Vaults under version control or backup, 
in case you mess up. Like most terminal commands, _it's got no undo feature._


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
